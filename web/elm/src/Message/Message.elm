module Message.Message exposing
    ( DomID(..)
    , DropTarget(..)
    , Message(..)
    , PipelinesSection(..)
    , VersionId
    , VersionToggleAction(..)
    , VisibilityAction(..)
    )

import Concourse exposing (DatabaseID)
import Concourse.Cli as Cli
import Concourse.Pagination exposing (Page)
import Routes exposing (StepID)
import StrictEvents


type Message
    = -- Top Bar
      FilterMsg String
    | FocusMsg
    | BlurMsg
      -- Pipeline
    | ToggleGroup Concourse.PipelineGroup
    | SetGroups (List String)
      -- Dashboard
    | DragStart String String
    | DragOver DropTarget
    | DragEnd
    | Tooltip String String
    | TooltipHd String String
      -- Resource
    | EditComment String
    | FocusTextArea
    | BlurTextArea
      -- Build
    | ScrollBuilds StrictEvents.WheelEvent
    | RevealCurrentBuildInHistory
    | SetHighlight String Int
    | ExtendHighlight String Int
      -- common
    | Hover (Maybe DomID)
    | Click DomID
    | GoToRoute Routes.Route
    | Scrolled StrictEvents.ScrollState


type DomID
    = ToggleJobButton
    | TriggerBuildButton
    | AbortBuildButton
    | RerunBuildButton
    | PreviousPageButton
    | NextPageButton
    | CheckButton Bool
    | EditButton
    | SaveCommentButton
    | ResourceCommentTextarea
    | FirstOccurrenceGetStepLabel StepID
    | StepState StepID
    | PinIcon
    | PinMenuDropDown String
    | PinButton VersionId
    | PinBar
    | PipelineStatusIcon PipelinesSection Concourse.PipelineIdentifier
    | PipelineCardPauseToggle PipelinesSection Concourse.PipelineIdentifier
    | TopBarFavoritedIcon DatabaseID
    | TopBarPauseToggle Concourse.PipelineIdentifier
    | VisibilityButton PipelinesSection Concourse.PipelineIdentifier
    | PipelineCardFavoritedIcon PipelinesSection DatabaseID
    | FooterCliIcon Cli.Cli
    | WelcomeCardCliIcon Cli.Cli
    | CopyTokenButton
    | SendTokenButton
    | CopyTokenInput
    | JobGroup Int
    | StepTab String Int
    | StepHeader String
    | StepSubHeader String Int
    | ShowSearchButton
    | ClearSearchButton
    | LoginButton
    | LogoutButton
    | UserMenu
    | PaginationButton Page
    | VersionHeader VersionId
    | VersionToggle VersionId
    | BuildTab Int String
    | PipelineWrapper Concourse.PipelineIdentifier
    | JobPreview PipelinesSection Concourse.JobIdentifier
    | HamburgerMenu
    | SideBarResizeHandle
    | SideBarTeam PipelinesSection String
    | SideBarPipeline PipelinesSection Concourse.PipelineIdentifier
    | SideBarFavoritedIcon DatabaseID
    | Dashboard
    | DashboardGroup String


type PipelinesSection
    = FavoritesSection
    | AllPipelinesSection


type VersionToggleAction
    = Enable
    | Disable


type VisibilityAction
    = Expose
    | Hide


type alias VersionId =
    Concourse.VersionedResourceIdentifier


type alias DatabaseID =
    Int


type DropTarget
    = Before String
    | After String
