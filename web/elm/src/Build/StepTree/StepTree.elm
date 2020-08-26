module Build.StepTree.StepTree exposing
    ( extendHighlight
    , finished
    , init
    , setHighlight
    , switchTab
    , toggleStep
    , toggleStepSubHeader
    , view
    )

import Ansi.Log
import Array exposing (Array)
import Assets
import Build.Models exposing (StepHeaderType(..))
import Build.StepTree.Models
    exposing
        ( HookedStep
        , MetadataField
        , Step
        , StepName
        , StepState(..)
        , StepTree(..)
        , StepTreeModel
        , TabFocus(..)
        , TabInfo
        , Version
        , finishTree
        , focusTabbed
        , isActive
        , map
        , mostSevereStepState
        , toggleSubHeaderExpanded
        , treeIsActive
        , updateAt
        , wrapHook
        , wrapMultiStep
        , wrapStep
        )
import Build.Styles as Styles
import Colors
import Concourse exposing (JsonValue(..))
import DateFormat
import Dict exposing (Dict)
import Duration
import HoverState
import Html exposing (Html)
import Html.Attributes exposing (attribute, class, classList, href, id, style, target)
import Html.Events exposing (onClick, onMouseEnter, onMouseLeave)
import Json.Encode
import List.Extra
import Message.Effects exposing (Effect(..), toHtmlID)
import Message.Message exposing (DomID(..), Message(..))
import Routes exposing (Highlight(..), StepID, showHighlight)
import StrictEvents
import Time
import Tooltip
import Url exposing (fromString)
import Views.DictView as DictView
import Views.Icon as Icon
import Views.Spinner as Spinner


init :
    Highlight
    -> Concourse.BuildResources
    -> Concourse.BuildPlan
    -> StepTreeModel
init hl resources buildPlan =
    case buildPlan.step of
        Concourse.BuildStepTask name ->
            initBottom hl Task buildPlan name

        Concourse.BuildStepArtifactInput name ->
            initBottom hl
                (\s ->
                    ArtifactInput { s | state = StepStateSucceeded }
                )
                buildPlan
                name

        Concourse.BuildStepCheck name ->
            initBottom hl Check buildPlan name

        Concourse.BuildStepGet name version ->
            initBottom hl
                (Get << setupGetStep resources name version)
                buildPlan
                name

        Concourse.BuildStepArtifactOutput name ->
            initBottom hl ArtifactOutput buildPlan name

        Concourse.BuildStepPut name ->
            initBottom hl Put buildPlan name

        Concourse.BuildStepSetPipeline name ->
            initBottom hl SetPipeline buildPlan name

        Concourse.BuildStepLoadVar name ->
            initBottom hl LoadVar buildPlan name

        Concourse.BuildStepAggregate plans ->
            initMultiStep hl resources buildPlan.id Aggregate plans

        Concourse.BuildStepInParallel plans ->
            initMultiStep hl resources buildPlan.id InParallel plans

        Concourse.BuildStepDo plans ->
            initMultiStep hl resources buildPlan.id Do plans

        Concourse.BuildStepAcross plan ->
            let
                ( values, plans ) =
                    plan.steps
                        |> List.unzip
            in
            initRootedMultiStep hl
                resources
                buildPlan
                (plan.vars |> String.join ", ")
                (\step substeps ->
                    Across
                        plan.vars
                        values
                        (plans |> List.map (planIsHighlighted hl))
                        step
                        (substeps
                            |> Array.map (map (\s -> { s | expanded = True }))
                        )
                )
                (plans |> Array.fromList)

        Concourse.BuildStepRetry plans ->
            initMultiStep hl
                resources
                buildPlan.id
                (Retry <| startingTab hl buildPlan.id (Array.toList plans))
                plans

        Concourse.BuildStepOnSuccess hookedPlan ->
            initHookedStep hl resources OnSuccess hookedPlan

        Concourse.BuildStepOnFailure hookedPlan ->
            initHookedStep hl resources OnFailure hookedPlan

        Concourse.BuildStepOnAbort hookedPlan ->
            initHookedStep hl resources OnAbort hookedPlan

        Concourse.BuildStepOnError hookedPlan ->
            initHookedStep hl resources OnError hookedPlan

        Concourse.BuildStepEnsure hookedPlan ->
            initHookedStep hl resources Ensure hookedPlan

        Concourse.BuildStepTry plan ->
            initWrappedStep hl resources Try plan

        Concourse.BuildStepTimeout plan ->
            initWrappedStep hl resources Timeout plan


planIsHighlighted : Highlight -> Concourse.BuildPlan -> Bool
planIsHighlighted hl plan =
    case hl of
        HighlightNothing ->
            False

        HighlightLine stepID _ ->
            planContainsID stepID plan

        HighlightRange stepID _ _ ->
            planContainsID stepID plan


planContainsID : StepID -> Concourse.BuildPlan -> Bool
planContainsID stepID plan =
    plan |> Concourse.mapBuildPlan .id |> List.member stepID


startingTab : Highlight -> String -> List Concourse.BuildPlan -> TabInfo
startingTab hl planID plans =
    let
        idx =
            case hl of
                HighlightNothing ->
                    Nothing

                HighlightLine stepID _ ->
                    plans |> List.Extra.findIndex (planContainsID stepID)

                HighlightRange stepID _ _ ->
                    plans |> List.Extra.findIndex (planContainsID stepID)
    in
    case idx of
        Nothing ->
            { id = planID, tab = 0, focus = Auto }

        Just tab ->
            { id = planID, tab = tab, focus = User }


initMultiStep :
    Highlight
    -> Concourse.BuildResources
    -> String
    -> (Array StepTree -> StepTree)
    -> Array Concourse.BuildPlan
    -> StepTreeModel
initMultiStep hl resources planId constructor plans =
    let
        inited =
            Array.map (init hl resources) plans

        trees =
            Array.map .tree inited

        selfFoci =
            Dict.singleton planId identity
    in
    { tree = constructor trees
    , foci =
        inited
            |> Array.map .foci
            |> Array.indexedMap wrapMultiStep
            |> Array.foldr Dict.union selfFoci
    , highlight = hl
    }


constructStep : Highlight -> Concourse.BuildPlan -> StepName -> Step
constructStep hl plan name =
    { id = plan.id
    , name = name
    , state = StepStatePending
    , log = Ansi.Log.init Ansi.Log.Cooked
    , error = Nothing
    , expanded =
        case hl of
            HighlightNothing ->
                False

            HighlightLine stepID _ ->
                List.member stepID (Concourse.mapBuildPlan .id plan)

            HighlightRange stepID _ _ ->
                List.member stepID (Concourse.mapBuildPlan .id plan)
    , version = Nothing
    , metadata = []
    , firstOccurrence = False
    , timestamps = Dict.empty
    , initialize = Nothing
    , start = Nothing
    , finish = Nothing
    }


initBottom :
    Highlight
    -> (Step -> StepTree)
    -> Concourse.BuildPlan
    -> StepName
    -> StepTreeModel
initBottom hl create plan name =
    { tree = constructStep hl plan name |> create
    , foci = Dict.singleton plan.id identity
    , highlight = hl
    }


initRootedMultiStep :
    Highlight
    -> Concourse.BuildResources
    -> Concourse.BuildPlan
    -> StepName
    -> (Step -> Array StepTree -> StepTree)
    -> Array Concourse.BuildPlan
    -> StepTreeModel
initRootedMultiStep hl resources plan stepName constructor plans =
    initMultiStep
        hl
        resources
        plan.id
        (constructStep hl plan stepName |> constructor)
        plans


initWrappedStep :
    Highlight
    -> Concourse.BuildResources
    -> (StepTree -> StepTree)
    -> Concourse.BuildPlan
    -> StepTreeModel
initWrappedStep hl resources create plan =
    let
        { tree, foci } =
            init hl resources plan
    in
    { tree = create tree
    , foci = Dict.map (always wrapStep) foci
    , highlight = hl
    }


initHookedStep :
    Highlight
    -> Concourse.BuildResources
    -> (HookedStep -> StepTree)
    -> Concourse.HookedPlan
    -> StepTreeModel
initHookedStep hl resources create hookedPlan =
    let
        stepModel =
            init hl resources hookedPlan.step

        hookModel =
            init hl resources hookedPlan.hook
    in
    { tree = create { step = stepModel.tree, hook = hookModel.tree }
    , foci =
        Dict.union
            (Dict.map (always wrapStep) stepModel.foci)
            (Dict.map (always wrapHook) hookModel.foci)
    , highlight = hl
    }


setupGetStep : Concourse.BuildResources -> StepName -> Maybe Version -> Step -> Step
setupGetStep resources name version step =
    { step
        | version = version
        , firstOccurrence = isFirstOccurrence resources.inputs name
    }


isFirstOccurrence : List Concourse.BuildResourcesInput -> StepName -> Bool
isFirstOccurrence resources step =
    case resources of
        [] ->
            False

        { name, firstOccurrence } :: rest ->
            if name == step then
                firstOccurrence

            else
                isFirstOccurrence rest step


finished : StepTreeModel -> StepTreeModel
finished root =
    { root | tree = finishTree root.tree }


toggleStep : StepID -> StepTreeModel -> ( StepTreeModel, List Effect )
toggleStep id root =
    ( updateAt id (map (\step -> { step | expanded = not step.expanded })) root
    , []
    )


toggleStepSubHeader : StepID -> Int -> StepTreeModel -> ( StepTreeModel, List Effect )
toggleStepSubHeader id i root =
    ( updateAt id (toggleSubHeaderExpanded i) root, [] )


switchTab : StepID -> Int -> StepTreeModel -> ( StepTreeModel, List Effect )
switchTab id tab root =
    ( updateAt id (focusTabbed tab) root, [] )


setHighlight : StepID -> Int -> StepTreeModel -> ( StepTreeModel, List Effect )
setHighlight id line root =
    let
        hl =
            HighlightLine id line
    in
    ( { root | highlight = hl }, [ ModifyUrl (showHighlight hl) ] )


extendHighlight : StepID -> Int -> StepTreeModel -> ( StepTreeModel, List Effect )
extendHighlight id line root =
    let
        hl =
            case root.highlight of
                HighlightNothing ->
                    HighlightLine id line

                HighlightLine currentID currentLine ->
                    if currentID == id then
                        if currentLine < line then
                            HighlightRange id currentLine line

                        else
                            HighlightRange id line currentLine

                    else
                        HighlightLine id line

                HighlightRange currentID currentLine _ ->
                    if currentID == id then
                        if currentLine < line then
                            HighlightRange id currentLine line

                        else
                            HighlightRange id line currentLine

                    else
                        HighlightLine id line
    in
    ( { root | highlight = hl }, [ ModifyUrl (showHighlight hl) ] )


view :
    { timeZone : Time.Zone, hovered : HoverState.HoverState }
    -> StepTreeModel
    -> Html Message
view session model =
    viewTree session model model.tree 0


viewTree :
    { timeZone : Time.Zone, hovered : HoverState.HoverState }
    -> StepTreeModel
    -> StepTree
    -> Int
    -> Html Message
viewTree session model tree depth =
    case tree of
        Task step ->
            viewStep model session depth step StepHeaderTask

        ArtifactInput step ->
            viewStep model session depth step (StepHeaderGet False)

        Check step ->
            viewStep model session depth step StepHeaderCheck

        Get step ->
            viewStep model session depth step (StepHeaderGet step.firstOccurrence)

        ArtifactOutput step ->
            viewStep model session depth step StepHeaderPut

        Put step ->
            viewStep model session depth step StepHeaderPut

        SetPipeline step ->
            viewStep model session depth step StepHeaderSetPipeline

        LoadVar step ->
            viewStep model session depth step StepHeaderLoadVar

        Try step ->
            viewTree session model step depth

        Across vars vals expanded step substeps ->
            viewStepWithBody model session depth step StepHeaderAcross <|
                (List.map2 Tuple.pair vals expanded
                    |> List.indexedMap
                        (\i ( vals_, expanded_ ) ->
                            ( vals_
                            , expanded_
                            , substeps |> Array.get i
                            )
                        )
                    |> List.filterMap
                        (\( vals_, expanded_, substep ) ->
                            case substep of
                                Nothing ->
                                    -- impossible, but need to get rid of the Maybe
                                    Nothing

                                Just substep_ ->
                                    Just ( vals_, expanded_, substep_ )
                        )
                    |> List.indexedMap
                        (\i ( vals_, expanded_, substep ) ->
                            let
                                keyVals =
                                    List.map2 Tuple.pair vars vals_
                            in
                            viewAcrossStepSubHeader model session step.id i keyVals expanded_ (depth + 1) substep
                        )
                )

        Retry tabInfo steps ->
            Html.div [ class "retry" ]
                [ Html.ul
                    (class "retry-tabs" :: Styles.retryTabList)
                    (Array.toList <| Array.indexedMap (viewRetryTab session tabInfo) steps)
                , case Array.get tabInfo.tab steps of
                    Just step ->
                        viewTree session model step depth

                    Nothing ->
                        -- impossible (bogus tab selected)
                        Html.text ""
                ]

        Timeout step ->
            viewTree session model step depth

        Aggregate steps ->
            Html.div [ class "aggregate" ]
                (Array.toList <| Array.map (viewSeq session model depth) steps)

        InParallel steps ->
            Html.div [ class "parallel" ]
                (Array.toList <| Array.map (viewSeq session model depth) steps)

        Do steps ->
            Html.div [ class "do" ]
                (Array.toList <| Array.map (viewSeq session model depth) steps)

        OnSuccess { step, hook } ->
            viewHooked session "success" model depth step hook

        OnFailure { step, hook } ->
            viewHooked session "failure" model depth step hook

        OnAbort { step, hook } ->
            viewHooked session "abort" model depth step hook

        OnError { step, hook } ->
            viewHooked session "error" model depth step hook

        Ensure { step, hook } ->
            viewHooked session "ensure" model depth step hook


viewAcrossStepSubHeader :
    StepTreeModel
    -> { timeZone : Time.Zone, hovered : HoverState.HoverState }
    -> StepID
    -> Int
    -> List ( String, JsonValue )
    -> Bool
    -> Int
    -> StepTree
    -> Html Message
viewAcrossStepSubHeader model session stepID subHeaderIdx keyVals expanded depth subtree =
    let
        state =
            mostSevereStepState subtree
    in
    Html.div
        [ classList
            [ ( "build-step", True )
            , ( "inactive", not <| isActive state )
            ]
        , style "margin-top" "10px"
        ]
        [ Html.div
            ([ class "header"
             , class "sub-header"
             , onClick <| Click <| StepSubHeader stepID subHeaderIdx
             , style "z-index" <| String.fromInt <| max (maxDepth - depth) 1
             ]
                ++ Styles.stepHeader state
            )
            [ Html.div
                [ style "display" "flex" ]
                [ viewAcrossStepSubHeaderLabels keyVals ]
            , Html.div
                [ style "display" "flex" ]
                [ viewStepStateWithoutTooltip state ]
            ]
        , if expanded then
            Html.div
                [ class "step-body"
                , class "clearfix"
                , style "padding-bottom" "0"
                ]
                [ viewTree session model subtree (depth + 1) ]

          else
            Html.text ""
        ]


viewAcrossStepSubHeaderLabels : List ( String, JsonValue ) -> Html Message
viewAcrossStepSubHeaderLabels keyVals =
    Html.div Styles.acrossStepSubHeaderLabel
        (keyVals
            |> List.concatMap
                (\( k, v ) ->
                    viewAcrossStepSubHeaderKeyValue k v
                )
        )


viewAcrossStepSubHeaderKeyValue : String -> JsonValue -> List (Html Message)
viewAcrossStepSubHeaderKeyValue key val =
    let
        keyValueSpan text =
            [ Html.span
                [ style "display" "inline-block"
                , style "margin-right" "10px"
                ]
                [ Html.span [ style "color" Colors.pending ]
                    [ Html.text <| key ++ ": " ]
                , Html.text text
                ]
            ]
    in
    case val of
        JsonString s ->
            keyValueSpan s

        JsonNumber n ->
            keyValueSpan <| String.fromFloat n

        JsonRaw v ->
            keyValueSpan <| Json.Encode.encode 0 v

        JsonArray l ->
            List.indexedMap
                (\i v ->
                    let
                        subKey =
                            key ++ "[" ++ String.fromInt i ++ "]"
                    in
                    viewAcrossStepSubHeaderKeyValue subKey v
                )
                l
                |> List.concat

        JsonObject o ->
            List.concatMap
                (\( k, v ) ->
                    let
                        subKey =
                            key ++ "." ++ k
                    in
                    viewAcrossStepSubHeaderKeyValue subKey v
                )
                o


viewRetryTab :
    { r | hovered : HoverState.HoverState }
    -> TabInfo
    -> Int
    -> StepTree
    -> Html Message
viewRetryTab session tabInfo idx step =
    viewTab session tabInfo idx (String.fromInt (idx + 1)) step


viewTab :
    { r | hovered : HoverState.HoverState }
    -> TabInfo
    -> Int
    -> String
    -> StepTree
    -> Html Message
viewTab { hovered } tabInfo tab label step =
    Html.li
        ([ classList
            [ ( "current", tabInfo.tab == tab )
            , ( "inactive", not <| treeIsActive step )
            ]
         , onMouseEnter <| Hover <| Just <| StepTab tabInfo.id tab
         , onMouseLeave <| Hover Nothing
         , onClick <| Click <| StepTab tabInfo.id tab
         ]
            ++ Styles.tab
                { isHovered = HoverState.isHovered (StepTab tabInfo.id tab) hovered
                , isCurrent = tabInfo.tab == tab
                , isStarted = treeIsActive step
                }
        )
        [ Html.text label ]


viewSeq : { timeZone : Time.Zone, hovered : HoverState.HoverState } -> StepTreeModel -> Int -> StepTree -> Html Message
viewSeq session model depth tree =
    Html.div [ class "seq" ] [ viewTree session model tree depth ]


viewHooked : { timeZone : Time.Zone, hovered : HoverState.HoverState } -> String -> StepTreeModel -> Int -> StepTree -> StepTree -> Html Message
viewHooked session name model depth step hook =
    Html.div [ class "hooked" ]
        [ Html.div [ class "step" ] [ viewTree session model step depth ]
        , Html.div [ class "children" ]
            [ Html.div [ class ("hook hook-" ++ name) ] [ viewTree session model hook depth ]
            ]
        ]


maxDepth : Int
maxDepth =
    10


viewStepWithBody :
    StepTreeModel
    -> { timeZone : Time.Zone, hovered : HoverState.HoverState }
    -> Int
    -> Step
    -> StepHeaderType
    -> List (Html Message)
    -> Html Message
viewStepWithBody model session depth { id, name, log, state, error, expanded, version, metadata, timestamps, initialize, start, finish } headerType body =
    Html.div
        [ classList
            [ ( "build-step", True )
            , ( "inactive", not <| isActive state )
            ]
        , attribute "data-step-name" name
        ]
        [ Html.div
            ([ class "header"
             , onClick <| Click <| StepHeader id
             , style "z-index" <| String.fromInt <| max (maxDepth - depth) 1
             ]
                ++ Styles.stepHeader state
            )
            [ Html.div
                [ style "display" "flex" ]
                [ viewStepHeaderLabel headerType id
                , Html.h3 [] [ Html.text name ]
                ]
            , Html.div
                [ style "display" "flex" ]
                [ viewVersion version
                , viewStepState
                    state
                    id
                    (viewDurationTooltip
                        initialize
                        start
                        finish
                        (showTooltip session <| StepState id)
                    )
                ]
            ]
        , if expanded then
            Html.div
                [ class "step-body"
                , class "clearfix"
                ]
                ([ viewMetadata metadata
                 , Html.pre [ class "timestamped-logs" ] <|
                    viewLogs log timestamps model.highlight session.timeZone id
                 , case error of
                    Nothing ->
                        Html.span [] []

                    Just msg ->
                        Html.span [ class "error" ] [ Html.pre [] [ Html.text msg ] ]
                 ]
                    ++ body
                )

          else
            Html.text ""
        ]


viewStep : StepTreeModel -> { timeZone : Time.Zone, hovered : HoverState.HoverState } -> Int -> Step -> StepHeaderType -> Html Message
viewStep model session depth step headerType =
    viewStepWithBody model session depth step headerType []


showTooltip : Tooltip.Model b -> DomID -> Bool
showTooltip session domID =
    case session.hovered of
        HoverState.Tooltip x _ ->
            x == domID

        _ ->
            False


viewLogs :
    Ansi.Log.Model
    -> Dict Int Time.Posix
    -> Highlight
    -> Time.Zone
    -> String
    -> List (Html Message)
viewLogs { lines } timestamps hl timeZone id =
    Array.toList <|
        Array.indexedMap
            (\idx line ->
                viewTimestampedLine
                    { timestamps = timestamps
                    , highlight = hl
                    , id = id
                    , lineNo = idx + 1
                    , line = line
                    , timeZone = timeZone
                    }
            )
            lines


viewTimestampedLine :
    { timestamps : Dict Int Time.Posix
    , highlight : Highlight
    , id : StepID
    , lineNo : Int
    , line : Ansi.Log.Line
    , timeZone : Time.Zone
    }
    -> Html Message
viewTimestampedLine { timestamps, highlight, id, lineNo, line, timeZone } =
    let
        highlighted =
            case highlight of
                HighlightNothing ->
                    False

                HighlightLine hlId hlLine ->
                    hlId == id && hlLine == lineNo

                HighlightRange hlId hlLine1 hlLine2 ->
                    hlId == id && lineNo >= hlLine1 && lineNo <= hlLine2

        ts =
            Dict.get lineNo timestamps
    in
    Html.tr
        [ classList
            [ ( "timestamped-line", True )
            , ( "highlighted-line", highlighted )
            ]
        , Html.Attributes.id <| id ++ ":" ++ String.fromInt lineNo
        ]
        [ viewTimestamp
            { id = id
            , lineNo = lineNo
            , date = ts
            , timeZone = timeZone
            }
        , viewLine line
        ]


viewLine : Ansi.Log.Line -> Html Message
viewLine line =
    Html.td [ class "timestamped-content" ]
        [ Ansi.Log.viewLine line
        ]


viewTimestamp :
    { id : String
    , lineNo : Int
    , date : Maybe Time.Posix
    , timeZone : Time.Zone
    }
    -> Html Message
viewTimestamp { id, lineNo, date, timeZone } =
    Html.a
        [ href (showHighlight (HighlightLine id lineNo))
        , StrictEvents.onLeftClickOrShiftLeftClick
            (SetHighlight id lineNo)
            (ExtendHighlight id lineNo)
        ]
        [ case date of
            Just d ->
                Html.td
                    [ class "timestamp" ]
                    [ Html.text <|
                        DateFormat.format
                            [ DateFormat.hourMilitaryFixed
                            , DateFormat.text ":"
                            , DateFormat.minuteFixed
                            , DateFormat.text ":"
                            , DateFormat.secondFixed
                            ]
                            timeZone
                            d
                    ]

            _ ->
                Html.td [ class "timestamp placeholder" ] []
        ]


viewVersion : Maybe Version -> Html Message
viewVersion version =
    Maybe.withDefault Dict.empty version
        |> Dict.map (always Html.text)
        |> DictView.view []


viewMetadata : List MetadataField -> Html Message
viewMetadata =
    List.map
        (\{ name, value } ->
            ( Html.text name
            , Html.pre []
                [ case fromString value of
                    Just _ ->
                        Html.a
                            [ href value
                            , target "_blank"
                            , style "text-decoration-line" "underline"
                            ]
                            [ Html.text value ]

                    Nothing ->
                        Html.text value
                ]
            )
        )
        >> List.map
            (\( key, value ) ->
                Html.tr []
                    [ Html.td (Styles.metadataCell Styles.Key) [ key ]
                    , Html.td (Styles.metadataCell Styles.Value) [ value ]
                    ]
            )
        >> Html.table Styles.metadataTable


viewStepStateWithoutTooltip : StepState -> Html Message
viewStepStateWithoutTooltip state =
    let
        attributes =
            [ style "position" "relative" ]
    in
    case state of
        StepStateRunning ->
            Spinner.spinner
                { sizePx = 14
                , margin = "7px"
                }

        StepStatePending ->
            Icon.icon
                { sizePx = 28
                , image = Assets.PendingIcon
                }
                (attribute "data-step-state" "pending"
                    :: Styles.stepStatusIcon
                    ++ attributes
                )

        StepStateInterrupted ->
            Icon.icon
                { sizePx = 28
                , image = Assets.InterruptedIcon
                }
                (attribute "data-step-state" "interrupted"
                    :: Styles.stepStatusIcon
                    ++ attributes
                )

        StepStateCancelled ->
            Icon.icon
                { sizePx = 28
                , image = Assets.CancelledIcon
                }
                (attribute "data-step-state" "cancelled"
                    :: Styles.stepStatusIcon
                    ++ attributes
                )

        StepStateSucceeded ->
            Icon.icon
                { sizePx = 28
                , image = Assets.SuccessCheckIcon
                }
                (attribute "data-step-state" "succeeded"
                    :: Styles.stepStatusIcon
                    ++ attributes
                )

        StepStateFailed ->
            Icon.icon
                { sizePx = 28
                , image = Assets.FailureTimesIcon
                }
                (attribute "data-step-state" "failed"
                    :: Styles.stepStatusIcon
                    ++ attributes
                )

        StepStateErrored ->
            Icon.icon
                { sizePx = 28
                , image = Assets.ExclamationTriangleIcon
                }
                (attribute "data-step-state" "errored"
                    :: Styles.stepStatusIcon
                    ++ attributes
                )


viewStepState : StepState -> StepID -> List (Html Message) -> Html Message
viewStepState state stepID tooltip =
    let
        attributes =
            [ onMouseLeave <| Hover Nothing
            , onMouseEnter <| Hover (Just (StepState stepID))
            , id <| toHtmlID <| StepState stepID
            , style "position" "relative"
            ]
    in
    case state of
        StepStateRunning ->
            Spinner.spinner
                { sizePx = 14
                , margin = "7px"
                }

        StepStatePending ->
            Icon.iconWithTooltip
                { sizePx = 28
                , image = Assets.PendingIcon
                }
                (attribute "data-step-state" "pending"
                    :: Styles.stepStatusIcon
                    ++ attributes
                )
                tooltip

        StepStateInterrupted ->
            Icon.iconWithTooltip
                { sizePx = 28
                , image = Assets.InterruptedIcon
                }
                (attribute "data-step-state" "interrupted"
                    :: Styles.stepStatusIcon
                    ++ attributes
                )
                tooltip

        StepStateCancelled ->
            Icon.iconWithTooltip
                { sizePx = 28
                , image = Assets.CancelledIcon
                }
                (attribute "data-step-state" "cancelled"
                    :: Styles.stepStatusIcon
                    ++ attributes
                )
                tooltip

        StepStateSucceeded ->
            Icon.iconWithTooltip
                { sizePx = 28
                , image = Assets.SuccessCheckIcon
                }
                (attribute "data-step-state" "succeeded"
                    :: Styles.stepStatusIcon
                    ++ attributes
                )
                tooltip

        StepStateFailed ->
            Icon.iconWithTooltip
                { sizePx = 28
                , image = Assets.FailureTimesIcon
                }
                (attribute "data-step-state" "failed"
                    :: Styles.stepStatusIcon
                    ++ attributes
                )
                tooltip

        StepStateErrored ->
            Icon.iconWithTooltip
                { sizePx = 28
                , image = Assets.ExclamationTriangleIcon
                }
                (attribute "data-step-state" "errored"
                    :: Styles.stepStatusIcon
                    ++ attributes
                )
                tooltip


viewStepHeaderLabel : StepHeaderType -> StepID -> Html Message
viewStepHeaderLabel headerType stepID =
    let
        eventHandlers =
            if headerType == StepHeaderGet True then
                [ onMouseLeave <| Hover Nothing
                , onMouseEnter <| Hover <| Just <| FirstOccurrenceGetStepLabel stepID
                ]

            else
                []
    in
    Html.div
        (id (toHtmlID <| FirstOccurrenceGetStepLabel stepID)
            :: Styles.stepHeaderLabel headerType
            ++ eventHandlers
        )
        [ Html.text <|
            case headerType of
                StepHeaderGet _ ->
                    "get:"

                StepHeaderPut ->
                    "put:"

                StepHeaderTask ->
                    "task:"

                StepHeaderCheck ->
                    "check:"

                StepHeaderSetPipeline ->
                    "set_pipeline:"

                StepHeaderLoadVar ->
                    "load_var:"

                StepHeaderAcross ->
                    "across:"
        ]


viewDurationTooltip : Maybe Time.Posix -> Maybe Time.Posix -> Maybe Time.Posix -> Bool -> List (Html Message)
viewDurationTooltip minit mstart mfinish tooltip =
    if tooltip then
        case ( minit, mstart, mfinish ) of
            ( Just initializedAt, Just startedAt, Just finishedAt ) ->
                let
                    initDuration =
                        Duration.between initializedAt startedAt

                    stepDuration =
                        Duration.between startedAt finishedAt
                in
                [ Html.div
                    [ style "position" "inherit"
                    , style "margin-left" "-500px"
                    ]
                    [ Html.div
                        Styles.durationTooltip
                        [ DictView.view []
                            (Dict.fromList
                                [ ( "initialization"
                                  , Html.text (Duration.format initDuration)
                                  )
                                , ( "step", Html.text (Duration.format stepDuration) )
                                ]
                            )
                        ]
                    ]
                , Html.div
                    Styles.durationTooltipArrow
                    []
                ]

            _ ->
                []

    else
        []
