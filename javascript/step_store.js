var Fluxxor = require("fluxxor");
var Immutable = require("immutable");

var constants = {
  SET_STEP_RUNNING: 'SET_STEP_RUNNING',
  SET_STEP_ERRORED: 'SET_STEP_ERRORED',
  SET_STEP_VERSION_INFO: 'SET_STEP_VERSION_INFO',
  TOGGLE_STEP_LOGS: 'TOGGLE_STEP_LOGS',
};

var Store = Fluxxor.createStore({
  initialize: function() {
    this.steps = Immutable.Map();

    this.bindActions(
      constants.SET_STEP_RUNNING, this.onSetStepRunning,
      constants.SET_STEP_ERRORED, this.onSetStepErrored,
      constants.SET_STEP_VERSION_INFO, this.onSetStepVersionInfo,
      constants.TOGGLE_STEP_LOGS, this.onToggleStepLogs
    );
  },

  setStep: function(origin, changes) {
    this.steps = this.steps.updateIn(origin.location, function(stepModel) {
      if (stepModel === undefined) {
        return new StepModel(origin).merge(changes);
      } else {
        return stepModel.merge(changes);
      }
    });

    this.emit("change");
  },

  onSetStepVersionInfo: function(data) {
    this.setStep(data.origin, { version: data.version, metadata: data.metadata });
  },

  onSetStepRunning: function(data) {
    this.setStep(data.origin, { running: data.running });
  },

  onSetStepErrored: function(data) {
    this.setStep(data.origin, { errored: data.errored });
  },

  onToggleStepLogs: function(data) {
    var step = this.steps.getIn(data.origin.location);
    this.setStep(data.origin, { showLogs: !step.isShowingLogs() });
  },

  getState: function() {
    return this.steps;
  },
});

function StepModel(origin) {
  this._map = new Immutable.Map({
    origin: origin,
    showLogs: origin.type == "execute",

    running: false,
    errored: false,

    version: undefined,
    metadata: undefined
  });

  this.merge = function(attrs) {
    var newModel = new StepModel(this.origin());
    newModel._map = this._map.merge(attrs);
    return newModel;
  }

  this.origin = function() {
    return this._map.get("origin");
  }

  this.isShowingLogs = function() {
    return this._map.get("showLogs");
  }

  this.isRunning = function() {
    return this._map.get("running");
  }

  this.isErrored = function() {
    return this._map.get("errored");
  }

  this.isFirstOccurrence = function() {
    // currently not supported
    return false;
  }

  this.version = function() {
    var x = this._map.get("version");
    if (x === undefined) {
      return undefined;
    }

    return x.toJS();
  }

  this.metadata = function() {
    var meta = this._map.get("metadata");
    if (meta === undefined) {
      return undefined;
    }

    return meta.toJS();
  }
}

module.exports = {
  Store: Store
};

for (var k in constants) {
  module.exports[k] = constants[k];
}
