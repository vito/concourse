package integration_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/db"
	"github.com/concourse/concourse/fly/rc"
	"github.com/concourse/concourse/skymarshal/token"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"gopkg.in/square/go-jose.v2/jwt"
)

var flyPath string
var homeDir string

var atcServer *ghttp.Server

const targetName = "testserver"
const teamName = "main"
const atcVersion = "4.0.0"
const workerVersion = "4.5.6"

var teams = []atc.Team{
	atc.Team{
		ID:   1,
		Name: "main",
	},
	atc.Team{
		ID:   2,
		Name: "other-team",
	},
}

var _ = SynchronizedBeforeSuite(func() []byte {
	binPath, err := gexec.Build("github.com/concourse/concourse/fly")
	Expect(err).NotTo(HaveOccurred())

	return []byte(binPath)
}, func(data []byte) {
	flyPath = string(data)

	SetDefaultEventuallyTimeout(10 * time.Second)
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})

func infoHandler() http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/api/v1/info"),
		ghttp.RespondWithJSONEncoded(200, atc.Info{Version: atcVersion, WorkerVersion: workerVersion}),
	)
}

func tokenHandler() http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("POST", "/sky/issuer/token"),
		ghttp.RespondWithJSONEncoded(
			200,
			oauthToken(),
		),
	)
}

func userInfoHandler() http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/api/v1/user"),
		ghttp.RespondWithJSONEncoded(200, map[string]interface{}{
			"user_name": "user",
			"teams": map[string][]string{
				teamName:          {"owner"},
				"some-team":       {"owner"},
				"some-other-team": {"owner"},
			},
		}),
	)
}

func validAccessToken(expiry time.Time) string {
	accessToken, err := token.Factory{}.GenerateAccessToken(db.Claims{
		Claims: jwt.Claims{Expiry: jwt.NewNumericDate(expiry)}},
	)
	if err != nil {
		panic(err)
	}
	return accessToken
}

func oauthToken() map[string]string {
	return map[string]string{
		"token_type":   "Bearer",
		"access_token": validAccessToken(time.Now()),
		"id_token":     "some-token",
	}
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func createFlyRc(targets rc.Targets) {
	flyrc := filepath.Join(homeDir, ".flyrc")

	flyrcBytes, err := json.Marshal(rc.RC{Targets: targets})
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(flyrc, flyrcBytes, 0600)
	if err != nil {
		panic(err)
	}
}

var _ = BeforeEach(func() {
	atcServer = ghttp.NewServer()

	atcServer.AppendHandlers(
		infoHandler(),
		tokenHandler(),
		userInfoHandler(),
		infoHandler(),
	)

	var err error

	homeDir, err = ioutil.TempDir("", "fly-test")
	Expect(err).NotTo(HaveOccurred())

	os.Setenv("HOME", homeDir)
	loginCmd := exec.Command(flyPath, "-t", targetName, "login", "-u", "user", "-p", "pass", "-c", atcServer.URL(), "-n", teamName)

	session, err := gexec.Start(loginCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	<-session.Exited

	Expect(session.ExitCode()).To(Equal(0))
})

var _ = AfterEach(func() {
	atcServer.Close()
	os.RemoveAll(homeDir)
})

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

func osFlag(short string, long string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("/%s, /%s", short, long)
	} else {
		return fmt.Sprintf("-%s, --%s", short, long)
	}
}

func userHomeDir() string {
	return os.Getenv("HOME")
}

func Change(fn func() int) *changeMatcher {
	return &changeMatcher{
		fn: fn,
	}
}

type changeMatcher struct {
	fn     func() int
	amount int

	before int
	after  int
}

func (cm *changeMatcher) By(amount int) *changeMatcher {
	cm.amount = amount

	return cm
}

func (cm *changeMatcher) Match(actual interface{}) (success bool, err error) {
	cm.before = cm.fn()

	ac, ok := actual.(func())
	if !ok {
		return false, errors.New("expected a function")
	}

	ac()

	cm.after = cm.fn()

	return (cm.after - cm.before) == cm.amount, nil
}

func (cm *changeMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected value to change by %d but it changed from %d to %d", cm.amount, cm.before, cm.after)
}

func (cm *changeMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected value not to change by %d but it changed from %d to %d", cm.amount, cm.before, cm.after)
}

const serverCert = `-----BEGIN CERTIFICATE-----
MIIC+DCCAeCgAwIBAgIRAK3uVYcWQA/y8Q8wHWnm0YgwDQYJKoZIhvcNAQELBQAw
EjEQMA4GA1UEChMHQWNtZSBDbzAeFw0xNjA4MDgyMzExMTFaFw0yNjA4MDYyMzEx
MTFaMBIxEDAOBgNVBAoTB0FjbWUgQ28wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAw
ggEKAoIBAQC+qY2Pfr79ltRLudcX45AyUPmOm0DwF2gE8HUihljtQmeWno5Gc2Uc
MqRrs7sfu90geL9ZBU7jYjhFxdlbsIO6710J0+uElLPKgSPI0sJDL3aoIi7jd+mi
qTyQ/OErOxtTOe7V3xUjAS9HrIcqVxKQFGwIic5sIOWhdg5zbVqoCI8eX5QHdxST
zNtoJYeCnCC5P7fdZySZ7lH5Y3HLgQWsVFyqoklKiYVmK1AyOQsZqrfOg1QjkXp9
xKN/Z0EsRsBGItvEnzQdVlaFFdo9yKnuWDzNvdwWJUpH/pdv6SOzvunAhZrNHe8w
DWUeLA6L5E8AvLR9KnT+BBCvygFu8njVAgMBAAGjSTBHMA4GA1UdDwEB/wQEAwIC
pDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MA8GA1UdEQQI
MAaHBH8AAAEwDQYJKoZIhvcNAQELBQADggEBAHPBHI8Vx/lD8KIPRBSfY2+XSXKQ
z4dHRFQxC4+hUm5X39Dg++ZgbHf5/Iv3T8466CW3DADCRamEpKmNK0/MAizDRmb2
sQ6qCVO5CrljEPgECY9MIV2MknbRIK6J0WhUEkTNY/RkGyLOkgGFD5Fadorf/b9D
0MKeDOl3xGIoDMz1qGS/ByiUXlu/5Dze3EKigtTI74z8GYIo/eAswfh3sIi0X7KR
vgkHttWh9tkfjV9IxuG/yCAnPTlCN7UI8YTZIH+SPqFakS8cIBzmVlOnZBsH4u2/
wtISX1uF4BH/i+knckhiG5mHNVSOVyUlZvC8lZR2hRMkeXVb/uns66Z/fSE=
-----END CERTIFICATE-----
`

const serverKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAvqmNj36+/ZbUS7nXF+OQMlD5jptA8BdoBPB1IoZY7UJnlp6O
RnNlHDKka7O7H7vdIHi/WQVO42I4RcXZW7CDuu9dCdPrhJSzyoEjyNLCQy92qCIu
43fpoqk8kPzhKzsbUznu1d8VIwEvR6yHKlcSkBRsCInObCDloXYOc21aqAiPHl+U
B3cUk8zbaCWHgpwguT+33Wckme5R+WNxy4EFrFRcqqJJSomFZitQMjkLGaq3zoNU
I5F6fcSjf2dBLEbARiLbxJ80HVZWhRXaPcip7lg8zb3cFiVKR/6Xb+kjs77pwIWa
zR3vMA1lHiwOi+RPALy0fSp0/gQQr8oBbvJ41QIDAQABAoIBAAzM9WQc7lW4Oqia
4YYJETVPmnGomsODzsgGHNckjfPf8XR7ULIKLU+nVsKkXnvS8RWtBavEX3eEsKJ+
lglB4JY8W9K9F6LfGPMPmIdzHvfDyAOhx+QduOHi2t4hHDz6yurbiN1zDMg83B/D
xY9iKSzjMh2gous/iis88dtuDBgb3RV903oiNJmTmHbZiClSEe9r3TWfOlxVMH0B
kFMvsnvRDomDzyfnjDTK+C8fPp07O3/uIM8rbOJaVEBYOVKj/pFlYA0HHY4g+sq5
zYSGzOLJLCVooU8hOYq3DuhYuFliziGDJZx3vg08GKVYwmcBaIlmYxPtFScyliKx
vRTFEgECgYEAwgj0ZUPA/DHyCtydwKUXjCV+j5uQJwDfes2qFDGbhcT6xkoGIM3S
EQl+Lu3NlRXJqZZfyZjurCuK9hOMIWK7Brlm/TyDV5CnOSK86/ez4mOL7mf802zZ
+aMqITebdj1BMLa3IGZhsw4hguLHQ7qelvJpyE/7531OEcyH6AB29MkCgYEA+4zf
BkW1PO7gSAZLU0RA5mkPjLV61OVrL3q7Yfq1sYC+dD/kQ7ug36ElZKLwtnPyPB2D
Yb0fxwDRCAeF2VZE58gVJwC1xtVhVI7DgXRgGdXZZq8EmW5/308mwLov2RfR/4lE
SgQ2gLruVZSt4hqXqmT2CV2UbKwDapDhTEC+Ja0CgYB8s5KWLjguHM9Iycac071R
dZtkIf9AAeCepOTEu6kPDKx6mYJcvMpf5rDw6iYwxWLomdsPzji97/IL+j4aCsDW
LnuRDr3+ndnK75dpM7WpLn71BmHHY3KnbISb+ofwMqfd7d+9c+8gS1mgK60SyzI3
Iq53bWgguzhcWg2SPhI1eQKBgQDJwJODwVb6NxDVU48Iip6O7kaVcVzB8ftEymgN
znn5kquuKyxWEt+VXPbTv0fW3imzg2xDcN9SydndWcNFrEZ5q+UjMhOZFL0Kh7JQ
WtlU/0ptbAQBVzniDeaj/vCvasZ38E1AHB7moobTRvsrdG6eMHmQy2hmvJPE3cyF
TwvyxQKBgEBP37kUkMg2D3JtnKJZX3r0DKf/3fvDXU+nsOipxF+QVi7SMt/30j2d
cGvJLpKX8qu9LOLGoVaB1yxEO8DO5B2YxdG/sjMBLlN+JSMFQ734VEIWD+1LiBvQ
JBf0Z15NGOY3w9KSxeiTPFyXU4/RmymDzyd/VKcnPBMKqTvbqT2G
-----END RSA PRIVATE KEY-----`
