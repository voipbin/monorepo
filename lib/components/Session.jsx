import React from 'react';
import PropTypes from 'prop-types';
import HangUpIcon from 'material-ui/svg-icons/communication/call-end';
import PauseIcon from 'material-ui/svg-icons/av/pause-circle-outline';
import ResumeIcon from 'material-ui/svg-icons/av/play-circle-outline';
import classnames from 'classnames';
import JsSIP from 'jssip';
import Logger from '../Logger';
import TransitionAppear from './TransitionAppear';

const logger = new Logger('Session');

export default class Session extends React.Component
{
	constructor(props)
	{
		super(props);

		this.state =
		{
			localHasVideo  : false,
			remoteHasVideo : false,
			localHold      : false,
			remoteHold     : false,
			canHold        : false,
			ringing        : false
		};

		// Mounted flag
		this._mounted = false;
		// Local cloned stream
		this._localClonedStream = null;
	}

	render()
	{
		const state = this.state;
		const props = this.props;
		let noRemoteVideo;

		if (props.session.isInProgress() && !state.ringing)
			noRemoteVideo = <div className='message'>connecting ...</div>;
		else if (state.ringing)
			noRemoteVideo = <div className='message'>ringing ...</div>;
		else if (state.localHold && state.remoteHold)
			noRemoteVideo = <div className='message'>both hold</div>;
		else if (state.localHold)
			noRemoteVideo = <div className='message'>local hold</div>;
		else if (state.remoteHold)
			noRemoteVideo = <div className='message'>remote hold</div>;
		else if (!state.remoteHasVideo)
			noRemoteVideo = <div className='message'>no remote video</div>;

		return (
			<TransitionAppear duration={1000}>
				<div data-component='Session'>
					<video
						ref='localVideo'
						className={classnames('local-video', { hidden: !state.localHasVideo })}
						autoPlay
						muted
					/>

					<div ref='remoteView' className='remote-view'></div>
					<div ref='remoteAudios' hidden></div>

					<video
						ref='remoteVideo'
						// className={classnames('remote-video', { hidden: noRemoteVideo })}
						className={classnames('remote-video', { hidden: true })}
						autoPlay
					/>


					<If condition={noRemoteVideo}>
						<div className='no-remote-video-info'>
							{noRemoteVideo}
						</div>
					</If>

					<div className='controls-container'>
						<div className='controls'>
							<HangUpIcon
								className='control'
								color={'#fff'}
								onClick={this.handleHangUp.bind(this)}
							/>

							<Choose>
								<When condition={!state.localHold}>
									<PauseIcon
										className='control'
										color={'#fff'}
										onClick={this.handleHold.bind(this)}
									/>
								</When>

								<Otherwise>
									<ResumeIcon
										className='control'
										color={'#fff'}
										onClick={this.handleResume.bind(this)}
									/>
								</Otherwise>
							</Choose>
						</div>
					</div>
				</div>
			</TransitionAppear>
		);
	}

	componentDidMount()
	{
		logger.debug('componentDidMount()');

		this._mounted = true;

		const localVideo = this.refs.localVideo;
		const session = this.props.session;
		const peerconnection = session.connection;
		const localStream = peerconnection.getLocalStreams()[0];
		const remoteStream = peerconnection.getRemoteStreams()[0];
		// const remoteStreams = new Map();

		window.remoteStreams = new Map();
		window.remoteVideos = new Map();
		window.remoteAudios = new Map();

		// Handle local stream
		if (localStream)
		{
			// Clone local stream
			this._localClonedStream = localStream.clone();

			// Display local stream
			localVideo.srcObject = this._localClonedStream;

			setTimeout(() =>
			{
				if (!this._mounted)
					return;

				if (localStream.getVideoTracks()[0])
					this.setState({ localHasVideo: true });
			}, 1000);
		}

		// If incoming all we already have the remote stream
		if (remoteStream)
		{
			logger.debug('already have a remote stream');

			this._handleRemoteStream(remoteStream);
		}

		if (session.isEstablished())
		{
			setTimeout(() =>
			{
				if (!this._mounted)
					return;

				this.setState({ canHold: true });
			});
		}

		session.on('progress', (data) =>
		{
			if (!this._mounted)
				return;

			logger.debug('session "progress" event [data:%o]', data);

			if (session.direction === 'outgoing')
				this.setState({ ringing: true });
		});

		session.on('accepted', (data) =>
		{
			if (!this._mounted)
				return;

			logger.debug('session "accepted" event [data:%o]', data);

			if (session.direction === 'outgoing')
			{
				this.props.onNotify(
					{
						level : 'success',
						title : 'Call answered'
					});
			}

			this.setState({ canHold: true, ringing: false });
		});

		session.on('failed', (data) =>
		{
			if (!this._mounted)
				return;

			logger.debug('session "failed" event [data:%o]', data);

			this.props.onNotify(
				{
					level   : 'error',
					title   : 'Call failed',
					message : `Cause: ${data.cause}`
				});

			if (session.direction === 'outgoing')
				this.setState({ ringing: false });
		});

		session.on('ended', (data) =>
		{
			if (!this._mounted)
				return;

			logger.debug('session "ended" event [data:%o]', data);

			this.props.onNotify(
				{
					level   : 'info',
					title   : 'Call ended',
					message : `Cause: ${data.cause}`
				});

			if (session.direction === 'outgoing')
				this.setState({ ringing: false });
		});

		session.on('hold', (data) =>
		{
			if (!this._mounted)
				return;

			const originator = data.originator;

			logger.debug('session "hold" event [originator:%s]', originator);

			switch (originator)
			{
				case 'local':
					this.setState({ localHold: true });
					break;
				case 'remote':
					this.setState({ remoteHold: true });
					break;
			}
		});

		session.on('unhold', (data) =>
		{
			if (!this._mounted)
				return;

			const originator = data.originator;

			logger.debug('session "unhold" event [originator:%s]', originator);

			switch (originator)
			{
				case 'local':
					this.setState({ localHold: false });
					break;
				case 'remote':
					this.setState({ remoteHold: false });
					break;
			}
		});

		peerconnection.addEventListener('track', (event) =>
		{
			logger.debug('peerconnection "track" event. event: %o', event);
			logger.debug('stream: %o', event.streams[0]);

			const remoteStreams = window.remoteStreams;
			const stream = event.streams[0]

			for (let stream of event.streams) {
				remoteStreams.set(stream.id, stream);
				logger.debug('Added new video stream. streams: %o', remoteStreams);
				this._handleRemoteStream(stream);
			}

			if (!this._mounted)
			{
				logger.error('_handleRemoteStream() | component not mounted');
				return;
			}

			this._updateMediaView();
		});
	}

	componentWillUnmount()
	{
		logger.debug('componentWillUnmount()');

		this._mounted = false;
		JsSIP.Utils.closeMediaStream(this._localClonedStream);
	}

	handleHangUp()
	{
		logger.debug('handleHangUp()');

		this.props.session.terminate();
	}

	handleHold()
	{
		logger.debug('handleHold()');

		this.props.session.hold({ useUpdate: true });
	}

	handleResume()
	{
		logger.debug('handleResume()');

		this.props.session.unhold({ useUpdate: true });
	}

	_handleRemoteStream(stream)
	{
		logger.debug('_handleRemoteTrack() [stream:%o]', stream);

		const remoteVideo = this.refs.remoteVideo;

		// Display remote stream
		remoteVideo.srcObject = stream;

		stream.addEventListener('addtrack', (event) =>
		{
			logger.debug('remote stream "addtrack" event [%o]', event);
			const track = event.track;

			// if (remoteVideo.srcObject !== stream)
			// 	return;

			// logger.debug('remote stream "addtrack" event [track:%o]', track);

			// // Refresh remote video
			// remoteVideo.srcObject = stream;

			// // this._checkRemoteVideo(stream);

			track.addEventListener('ended', () =>
			{
				logger.debug('remote track "ended" event [track:%o]', track);
			});

			this._updateMediaView();
		});

		stream.addEventListener('removetrack', () =>
		{
			logger.debug('remote stream "removetrack" event');

			// if (remoteVideo.srcObject !== stream)
			// 	return;

			// // Refresh remote video
			// remoteVideo.srcObject = stream;

			// // this._checkRemoteVideo(stream);
			this._updateMediaView();
		});

	}

	_handleRemoteMediaStream(stream)
	{
		logger.debug('_handleRemoteStream() [stream:%o]', stream);

		const remoteVideo = this.refs.remoteVideo;

		// Display remote stream
		remoteVideo.srcObject = stream;

		// this._checkRemoteVideo(stream);

		stream.addEventListener('addtrack', (event) =>
		{
			logger.debug('remote stream "addtrack" event [%o]', event);
			const track = event.track;

			if (remoteVideo.srcObject !== stream)
				return;

			logger.debug('remote stream "addtrack" event [track:%o]', track);

			// Refresh remote video
			remoteVideo.srcObject = stream;

			// this._checkRemoteVideo(stream);

			track.addEventListener('ended', () =>
			{
				logger.debug('remote track "ended" event [track:%o]', track);
			});
		});

		stream.addEventListener('removetrack', () =>
		{
			if (remoteVideo.srcObject !== stream)
				return;

			logger.debug('remote stream "removetrack" event');

			// Refresh remote video
			remoteVideo.srcObject = stream;

			// this._checkRemoteVideo(stream);
		});

	}

	_checkRemoteMedia()
	{
		const streams = window.remoteStreams;

		if (!this._mounted) {
			logger.error('_checkRemoteVideo() | component not mounted');
			return;
		}

		if (streams.size > 0) {
			this.setState({ remoteHasVideo: true });
		}
		else {
			this.setState({ remoteHasVideo: false });
		}
	}

	_calcWidth(count) {
		if (count <= 1) {
			return '100%';
		}
		else if (count == 2) {
			return '50%';
		}
		else if (count == 3 || count == 4) {
			return '45%';
		}
		else {
			return '25%';
		}
	}

	_calcHeight(count) {
		if (count < 4) {
			return '100%';
		}
		else {
			return '30%';
		}
	}

	_updateMediaView()
	{
		const streams = window.remoteStreams;
		const audiosStreams = window.remoteAudios;

		this._checkRemoteMedia()

		const width = this._calcWidth(streams.size);
		const height = this._calcHeight(streams.size);

		logger.debug('updateMediaView. streams: %o', streams);
		const remoteView = this.refs.remoteView;
		remoteView.innerHTML = '';

		// audios
		const remoteAudios = this.refs.remoteAudios;
		remoteAudios.innerHTML = '';

		// videos
		for (let [key, stream] of streams) {
			logger.debug('update remote video stream stream. %o: %o', key, stream);

			let video = document.createElement("video");
			video.style.width = width;
			// video.style.height = height;
			video.autoplay = true;
			video.srcObject = stream;

			// videos
			let tracks = stream.getVideoTracks();
			for (let i = 0; i < tracks.length; i++) {
				tracks[i].enabled = true;
			}

			// audio
			let audio = document.createElement("audio");
			audio.autoplay = true;
			audio.srcObject = stream;
			tracks = stream.getAudioTracks();
			for (let i = 0; i < tracks.length; i++) {
				tracks[i].enabled = true;
			}

			remoteView.appendChild(video);
			remoteAudios.appendChild(audio);
		}

	}

}

Session.propTypes =
{
	session            : PropTypes.object.isRequired,
	onNotify           : PropTypes.func.isRequired,
	onHideNotification : PropTypes.func.isRequired
};
