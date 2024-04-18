import clone from 'clone';
import deepmerge from 'deepmerge';
import Logger from './Logger';

import storage from './storage';

const logger = new Logger('settingsManager');

const DEFAULT_SIP_DOMAIN = 'conference.voipbin.net';
// const DEFAULT_SIP_DOMAIN = '192.168.56.10';
const DEFAULT_SETTINGS =
{
	'display_name': null,
	uri: null,
	password: null,
	destination: null,
	'api_username': null,
	'api_password': null,
	'api_token': null,
	socket:
	{
		uri: 'wss://conference.voipbin.net',
		// uri             : 'wss://192.168.56.10:8089/ws',
		'via_transport': 'auto'
	},
	'registrar_server': null,
	'contact_uri': null,
	'authorization_user': null,
	'instance_id': null,
	'session_timers': true,
	'use_preloaded_route': false,
	pcConfig:
	{
		// bundlePolicy: "max-bundle",
		// bundlePolicy: "max-compat",
		iceCandidatePoolSize: 1,
		rtcpMuxPolicy: 'require',
		iceServers:
			[
				{ urls: ['stun:stun.l.google.com:19302'] }
				// { urls: 'stun:stun.softjoys.com' }
				// { urls: [ 'stun:stun.stunprotocol.org:3478' ] }
				// { urls: [ 'stun:stun.voipstunt.com' ] }
			]
		// iceTransportPolicy: 'relay',
		// rtcpMuxPolicy : 'negotiate',
	},
	callstats:
	{
		enabled: false,
		AppID: null,
		AppSecret: null
	}
};

let settings;

// // First, read settings from local storage
// settings = storage.get();

if (settings)
	logger.debug('settings found in local storage');

// Try to read settings from a global SETTINGS object
if (window.SETTINGS) {
	logger.debug('window.SETTINGS found');

	settings = deepmerge(
		window.SETTINGS,
		settings || {},
		{ arrayMerge: (destinationArray, sourceArray) => sourceArray });
}

// If not settings are found, clone default ones
if (!settings) {
	logger.debug('no settings found, using default ones');

	settings = clone(DEFAULT_SETTINGS, false);
}
logger.debug('use setting. %s', JSON.stringify(settings));

module.exports =
{
	get() {
		return settings;
	},

	set(newSettings) {
		storage.set(newSettings);
		settings = newSettings;
	},

	clear() {
		storage.clear();
		settings = clone(DEFAULT_SETTINGS, false);
	},

	isReady() {
		return Boolean(settings.uri);
	},

	getDefaultDomain() {
		return DEFAULT_SIP_DOMAIN;
	}
};
