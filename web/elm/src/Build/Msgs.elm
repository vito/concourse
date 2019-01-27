module Build.Msgs exposing (HoveredButton(..), Msg(..), StepID)

import Concourse
import Concourse.BuildEvents
import Keyboard
import Scroll
import StrictEvents
import Time


type Msg
    = Noop
    | SwitchToBuild Concourse.Build
    | Hover HoveredButton
    | TriggerBuild (Maybe Concourse.JobIdentifier)
    | AbortBuild Int
    | ScrollBuilds StrictEvents.MouseWheelEvent
    | ClockTick Time.Time
    | RevealCurrentBuildInHistory
    | WindowScrolled Scroll.FromBottom
    | NavTo String
    | NewCSRFToken String
    | KeyPressed Keyboard.KeyCode
    | KeyUped Keyboard.KeyCode
    | BuildEventsMsg Concourse.BuildEvents.Msg
    | ToggleStep String
    | SwitchTab String Int
    | SetHighlight String Int
    | ExtendHighlight String Int
    | ScrollDown


type alias StepID =
    String


type HoveredButton
    = Neither
    | Abort
    | Trigger
    | FirstOccurrence StepID
