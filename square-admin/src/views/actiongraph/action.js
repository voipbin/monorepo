
const emptyID = "00000000-0000-0000-0000-000000000000";
const startID = "00000000-0000-0000-0000-000000000001";
var actions = [];

export function getActions() {
  return actions;
}

export function resetActions() { 
  console.log("Resetting actions.");
  actions = []; 
}

export function addAction(id, type, nextID, option) {

  const action = {
    "id": id,
    "type": type,
    "option": option,
    "next_id": nextID,
  };

  actions = [...actions, action];

  console.log("addAction. action: %v, actions_all: %v", action, actions);
}

export function updateAction(id, type, option) {

  for(let i = 0; i < actions.length; i++) {
    let action = actions[i];

    if(action.ID == id) {
      action.type = type;
      action.option = option;
      break;
    }
  }

  console.log("updateAction. actions_all: %o", actions);
}

export function removeAction(id) {

  for(let i = 0; i < actions.length; i++) {
    let action = actions[i];
    if(action.id == id) {
      actions.splice(i, 1);
    }
  }

  console.log("removeAction. actions_all: %o", actions);
}

export function connectAction(sourceID, sourceHandleID, targetID, targetHandleID) {
  console.log("connectAction. source_id: %s, source_handle_id: %s, target_id: %s, target_handle_id: %s", sourceID, sourceHandleID, targetID, targetHandleID);
  
  // if the source is the start,
  // we just move the given target to the front of the actions.
  if (sourceID == startID) {
    console.log("The flow's start point has changed. target_id: %s", targetID);
    // move the given target action to the front
    moveActionToFront(targetID);
    return;
  }

  const i = getActionIdx(sourceID);
  if (isNaN(i)) {
    console.log("Could not find action index. source_id: %s, target_id: %s", sourceID, targetID);
    return;
  }

  if (actions[i].type == "branch") {

    const tmpID = actions[i].id + "-source_";
    if (sourceHandleID.startsWith(tmpID)) {
      // 
      console.log("connectAction for target id connect");
      const idx = Number(sourceHandleID.slice(tmpID.length));
      console.log("Found key index. idx: %d", idx);

      const key = Object.keys(actions[i].option.target_ids)[idx];
      actions[i].option.target_ids[key] = targetID;
      console.log("Update action. action: %o", actions[i]);
    } else {
      console.log("connectAction for default target id. source_handle_id: %s", sourceHandleID);
      actions[i].option.default_target_id = targetID;
    }
  } else if (actions[i].type == "goto") {
    if (sourceHandleID == actions[i].id + "-source_target") {
      actions[i].option.target_id = targetID;
    } else {
      actions[i]["next_id"] = targetID;
    }
  } else {
    console.log("connectAction. idx: %d, action: %o, action_all: %o", i, actions[i], actions);
    actions[i]["next_id"] = targetID;
  }
}

export function disconnectAction(sourceID) {
  console.log("disconnectAction. source_id: %s", sourceID);
  
  if (sourceID == startID) {
    console.log("The flow's start point has removed. target_id: %s", sourceID);
    return;
  }

  const i = getActionIdx(sourceID);
  if (isNaN(i)) {
    console.log("Could not find action index. source_id: %s, target_id: %s", sourceID);
    return;
  }

  actions[i]["next_id"] = emptyID;
  console.log("disconnectAction. action_all: %o", actions);
}


function getActionIdx(id) {
  for(let i = 0; i < actions.length; i++) {
    if (actions[i].id == id) {
      return i;
    }
  }

  console.log("Could not find action index. id: %s, actions: %o", id, actions);
}

// moveActionToFront moves the given actionID to the front of the actions.
// this is required when the start node's edge has been changed.
function moveActionToFront(actionID) {
  console.log("moveActionToFront. action: %s, actions_all: %o", actionID, actions);
  const i = getActionIdx(actionID);
  if (i == NaN) {
    console.log("Could not find action index. action_id: %s", actionID);
    return;
  }

  const item = actions[i];
  console.log("moveActionToFront found action. action: %o", item);


  actions.splice(i, 1);
  // delete actions[i];
  console.log("Move actino to the front. action: %o", item);

  actions = [item, ...actions]
  console.log("moveActionToFront all action. actions_all: %o", actions);
}