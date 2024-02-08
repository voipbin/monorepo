import { useCallback } from 'react';
import * as JsSIP from "jssip";
import {
  Get as ProviderGet,
} from './provider';
import { useSelector, useDispatch } from 'react-redux'
import { CurrentcallSetSession, CurrentcallSetStatus } from 'src/store';


var phone = JsSIP.UA;
var call = {
  'session': {},
  'id': '',
  'direction': '',
  'status': 'finished',
  'from': '',
  'to': ''
};
var local_audio = new Audio();

const fileRing = require("./assets/sounds/ringtone_iphone.mp3");
const fileDial = require("./assets/sounds/ringtone_telephone.mp3");

export const NewPhone = (dispatch) => {

  const agent = JSON.parse(localStorage.getItem("agent_info"));

  const addresses = agent["addresses"];
  console.log("addresses info. addresses: " + addresses);

  let registAddress = "";
  addresses.forEach(address => {
    console.log("address info. address: " + address);
    if (address.type == "extension") {
      registAddress = address;
    }
  });

  if (registAddress == "") {
    return
  }

  // get extension
  const target = "extensions/" + registAddress["target"];
  ProviderGet(target).then(result => {
    console.log("Detail extension. extension: " + result);

    // set extension
    const tmp = JSON.stringify(result);
    localStorage.setItem("extension_info", tmp);

    // init phone
    const username = result["username"];
    const password = result["password"];
    const registDomain = agent["customer_id"] + ".registrar.voipbin.net";
    const uri = "sip:" + username + "@" + registDomain;
    console.log("Registering phone. uri: ", uri, ", password: ", password);

    JsSIP.debug.enable('JsSIP:*');

    const websocketDomain = 'wss://' + registDomain
    var socket = new JsSIP.WebSocketInterface(websocketDomain);
    var configuration = {
      sockets: [socket],
      uri: uri,
      realm: registDomain,
      authorization_user: result["extension"],
      password: password
    };

    phone = new JsSIP.UA(configuration);
    console.log("Starting phone. phone: ", phone, ", configuration: ", configuration);

    phone.on('connecting', ua_on_connecting);
    phone.on('connected', ua_on_connected);
    phone.on('disconnected', ua_on_disconnected);
    phone.on('registered', ua_on_registered);
    phone.on('unregistered', ua_on_unregistered )
    phone.on('registrationFailed', ua_on_registrationfailed )
    phone.on('registrationExpiring', ua_on_registrationexpiring )
    phone.on('newRTCSession', ua_on_newRTCSession )
    phone.on('newMessage', ua_on_newmessage )
    phone.on('newOptions', ua_on_newOptions )
    phone.on('sipEvent', ua_on_sipEvent )

    phone.start();
  });

  const ua_on_newRTCSession = (e) => {
    console.log("Fired ua_on_newRTCSession. " + e.session);

    // set call
    set_call(e.session);
    console.log("Call" + call);

    console.log("Setting session. session: ", e.session);
    dispatch(CurrentcallSetSession(e.session));
  }

  // ua event listeners
  const ua_on_connecting = (e) => {
    console.log("Fired ua_on_connecting. event: ", e);
  }

  const ua_on_connected = (e) => {
    console.log("Fired ua_on_connected. event: ", e);
  }

  const ua_on_disconnected = (e) => {
    console.log("Fired ua_on_disconnected. event: ", e);
  }

  const ua_on_registered = (e) => {
    console.log("Fired ua_on_registered. event: ", e);
  }

  const ua_on_unregistered = (e) => {
    console.log("Fired ua_on_unregistered. event: ", e);
    phone.register();
  }

  const ua_on_registrationfailed = (e) => {
    console.log("Fired ua_on_registrationfailed. event: ", e);
  }

  const ua_on_registrationexpiring = (e) => {
    console.log("Fired ua_on_registrationexpiring. event: ", e);
    phone.register();
  }

  const ua_on_newmessage = (e) => {
    console.log("Fired ua_on_newmessage. event: ", e);
  }

  const ua_on_newOptions = (e) => {
    console.log("Fired ua_on_newOptions. event: ", e);
  }

  const ua_on_sipEvent = (e) => {
    console.log("Fired ua_on_sipEvent. event: ", e);
  }

  const set_call = (s) => {
    call['session'] = s;
    call['id'] = s.id;
    call['direction'] = s.direction;

    if (call['direction'] == 'incoming') {
      // incoming
      call['from'] = s.remote_identity.uri.user;
      call['to'] = s.local_identity.uri.user;

      call['status'] = 'ringing';
      audio_play_src(fileRing, true);

    } else {
      // outgoing
      call['from'] = s.local_identity.uri.user;
      call['to'] = s.remote_identity.uri.user;

      call['status'] = 'dialing';
      audio_play_src(fileDial, true);
    }

    // set event handlers
    s.on('peerconnection', session_on_peerconnection);
    s.on('connecting', session_on_connecting);
    s.on('sending', session_on_sending);
    s.on('progress', session_on_progress);
    s.on('accepted', session_on_accepted);
    s.on('confirmed', session_on_confirmed);
    s.on('ended', session_on_ended);
    s.on('failed', session_on_failed);
    s.on('newDTMF', session_on_newDTMF);
    s.on('newInfo', session_on_newInfo);
    s.on('hold', session_on_hold);
    s.on('unhold', session_on_unhold);
    s.on('muted', session_on_muted);
    s.on('unmuted', session_on_unmuted);
    s.on('reinvite', session_on_reinvite);
    s.on('update', session_on_update);
    s.on('refer', session_on_refer);
    s.on('replaces', session_on_replaces);
    s.on('sdp', session_on_sdp);
    s.on('getusermediafailed', session_on_getusermediafailed);

    console.log("Set session. call: ", call)
  }

  // session

  const session_on_peerconnection = (e) => {
    console.log("Fired session_on_peerconnection. session: " + call["session"] + ", event: " + e);
  }

  const session_on_connecting = (e) => {
    console.log("Fired session_on_connecting. session: " + call["session"] + ", event: " + e);
  }

  const session_on_sending = (e) => {
    console.log("Fired session_on_sending. session: " + call["session"] + ", event: " + e);
  }

  const session_on_progress = (e) => {
    console.log("Fired session_on_progress. session: " + call["session"] + ", event: " + e);

    if (call["status"] == "ringing" || call["status"] == "progress") {
      return;
    }

    call["status"] = "ringing";
    dispatch(CurrentcallSetStatus("ringing"));

    // set remote audio
    if (call["session"].connection.getRemoteStreams().length > 0) {
      console.log("Setting remote stream.")
      audio_play_stream(call["session"].connection.getRemoteStreams()[0]);
    }
  }

  const session_on_accepted = (e) => {
    console.log("Fired session_on_accepted. session: " + call["session"] + ", event: " + e);
  }

  const session_on_confirmed = (e) => {
    console.log("Fired session_on_confirmed. session: " + call["session"] + ", event: " + e);

    call['status'] = 'progress';
    dispatch(CurrentcallSetStatus("progress"));

    if (call["session"].connection.getRemoteStreams().length > 0) {
      console.log("Setting remote stream.")
      audio_play_stream(call["session"].connection.getRemoteStreams()[0]);
    }
  }

  const session_on_ended = (e) => {
    console.log("Fired session_on_ended. session: " + call["session"] + ", event: " + e);

    call["status"] = "finished";
    dispatch(CurrentcallSetStatus("finished"));

    audio_stop();
  }

  const session_on_failed = (e) => {
    console.log("Fired session_on_failed. session: " + call["session"] + ", event: " + e);

    call["status"] = "finished";
    dispatch(CurrentcallSetStatus("finished"));

    audio_stop();
  }

  const session_on_newDTMF = (e) => {
    console.log("Fired session_on_newDTMF. session: " + call["session"] + ", event: " + e);
  }

  const session_on_newInfo = (e) => {
    console.log("Fired session_on_newInfo. session: " + call["session"] + ", event: " + e);
  }

  const session_on_hold = (e) => {
    console.log("Fired session_on_hold. session: " + call["session"] + ", event: " + e);
  }

  const session_on_unhold = (e) => {
    console.log("Fired session_on_unhold. session: " + call["session"] + ", event: " + e);
  }

  const session_on_muted = (e) => {
    console.log("Fired session_on_muted. session: " + call["session"] + ", event: " + e);
  }

  const session_on_unmuted = (e) => {
    console.log("Fired session_on_unmuted. session: " + call["session"] + ", event: " + e);
  }

  const session_on_reinvite = (e) => {
    console.log("Fired session_on_reinvite. session: " + call["session"] + ", event: " + e);
  }

  const session_on_update = (e) => {
    console.log("Fired session_on_update. session: " + call["session"] + ", event: " + e);
  }

  const session_on_refer = (e) => {
    console.log("Fired session_on_refer. session: " + call["session"] + ", event: " + e);
  }

  const session_on_replaces = (e) => {
    console.log("Fired session_on_replaces. session: " + call["session"] + ", event: " + e);
  }

  const session_on_sdp = (e) => {
    console.log("Fired session_on_sdp. session: " + call["session"] + ", event: " + e);
  }

  const session_on_getusermediafailed = (e) => {
    console.log("Fired session_on_getusermediafailed. session: " + call["session"] + ", event: " + e);
  }

}

export const CallDial = (destination) => {
  console.log("CallDial. destination: ", destination);

  const s = phone.call(
    destination,
    {
      mediaConstraints: {
        'audio': true,
        'video': false
      },
      rtcOfferConstraints: {
        offerToReceiveAudio: 1,
        offerToReceiveVideo: 0
      }
    }
  );

  console.log("Dialing the call. session: ", s)
}

export const CallAnswer = () => {
  console.log("CallAnswer. call", call);

  if (call["status"] != "ringing") {
    return;
  }

  call["session"].answer();
}

export const CallHangup = () => {
  console.log("CallHangupaa. call", call);

  if (call["status"] == "finished") {
    return;
  }

  audio_stop();
  call["session"].terminate();
}

export const CallGetInfo = () => {
  return call;
}

const audio_play_stream = (stream) =>{
  console.log("Fired set_remote_audio_stream")

  audio_stop();

  local_audio = new Audio();
  local_audio.srcObject = stream;
  local_audio.volume = 1.0;
  local_audio.play();
}

const audio_play_src =(src, repeat=false)=> {
  console.log("Fired set_remote_audio_src. src: ", src)

  audio_stop();

  local_audio = new Audio();
  local_audio.src = src;
  local_audio.volume = 1.0;
  local_audio.play();

  if(repeat == true) {
    local_audio.addEventListener('ended', function() {
      local_audio.play();
    }, false);
  }
}

const audio_stop = ()=> {
  local_audio.pause();
  local_audio.currentTime = 0;
}
