module Build.Models exposing
    ( Highlight(..)
    , HookedStep
    , MetadataField
    , Model
    , OutputModel
    , OutputState(..)
    , Page(..)
    , Step
    , StepFocus
    , StepHeaderType(..)
    , StepName
    , StepState(..)
    , StepTree(..)
    , StepTreeModel
    , TabFocus(..)
    , Version
    )

import Ansi.Log
import Array exposing (Array)
import Build.Msgs exposing (HoveredButton, Msg, StepID)
import Concourse
import Date exposing (Date)
import Dict exposing (Dict)
import Focus exposing (Focus)
import RemoteData exposing (WebData)
import Time exposing (Time)



-- Top level build


type alias Model =
    { page : Page
    , now : Maybe Time
    , job : Maybe Concourse.Job
    , history : List Concourse.Build
    , currentBuild : WebData CurrentBuild
    , browsingIndex : Int
    , autoScroll : Bool
    , csrfToken : String
    , previousKeyPress : Maybe Char
    , previousTriggerBuildByKey : Bool
    , showHelp : Bool
    , hash : String
    , hoveredButton : HoveredButton
    }


type alias CurrentBuild =
    { build : Concourse.Build
    , prep : Maybe Concourse.BuildPrep
    , output : Maybe OutputModel
    }


type Page
    = BuildPage Int
    | JobBuildPage Concourse.JobBuildIdentifier


type StepHeaderType
    = StepHeaderPut
    | StepHeaderGet Bool
    | StepHeaderTask



-- Output


type alias OutputModel =
    { steps : Maybe StepTreeModel
    , errors : Maybe Ansi.Log.Model
    , state : OutputState
    , eventSourceOpened : Bool
    , events : Sub Msg
    , highlight : Highlight
    }


type OutputState
    = StepsLoading
    | StepsLiveUpdating
    | StepsComplete
    | NotAuthorized



-- StepTree


type alias StepTreeModel =
    { tree : StepTree
    , foci : Dict StepID StepFocus
    , finished : Bool
    , highlight : Highlight
    , tooltip : Maybe StepID
    }


type StepTree
    = Task Step
    | Get Step
    | Put Step
    | Aggregate (Array StepTree)
    | Do (Array StepTree)
    | OnSuccess HookedStep
    | OnFailure HookedStep
    | OnAbort HookedStep
    | Ensure HookedStep
    | Try StepTree
    | Retry StepID Int TabFocus (Array StepTree)
    | Timeout StepTree


type alias StepFocus =
    Focus StepTree StepTree


type Highlight
    = HighlightNothing
    | HighlightLine StepID Int
    | HighlightRange StepID Int Int


type alias Step =
    { id : StepID
    , name : StepName
    , state : StepState
    , log : Ansi.Log.Model
    , error : Maybe String
    , expanded : Maybe Bool
    , version : Maybe Version
    , metadata : List MetadataField
    , firstOccurrence : Bool
    , timestamps : Dict Int Date
    }


type alias StepName =
    String


type StepState
    = StepStatePending
    | StepStateRunning
    | StepStateSucceeded
    | StepStateFailed
    | StepStateErrored


type alias Version =
    Dict String String


type alias MetadataField =
    { name : String
    , value : String
    }


type alias HookedStep =
    { step : StepTree
    , hook : StepTree
    }


type TabFocus
    = Auto
    | User
