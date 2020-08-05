import React from 'react';
import PropTypes from 'prop-types';
import TextField from 'material-ui/TextField';
import RaisedButton from 'material-ui/RaisedButton';
import classnames from 'classnames';
import Logger from '../Logger';
import utils from '../utils';
import UserChip from './UserChip';

const logger = new Logger('Dialer');

export default class Dialer extends React.Component {
	constructor(props) {
		super(props);

		this.state =
		{
			uri: props.callme || ''
		};
	}

	render() {
		const state = this.state;
		const props = this.props;
		const settings = props.settings;

		return (
			<div data-component='Dialer'>
				<div className='userchip-container'>
					<UserChip
						name={settings.display_name}
						uri={settings.uri || ''}
						status={props.status}
						fullWidth
					/>
				</div>

				<form
					className={classnames('uri-form', { hidden: props.busy && utils.isMobile() })}
					action=''
					onSubmit={this.handleSubmit.bind(this)}
				>
					<div className='uri-container'>
						<RaisedButton
							label='Start a new conference'
							primary
							disabled={!settings.api_token}
							onClick={this.handleCreateConference.bind(this)}
						/>
					</div>

					<div className='uri-container'>
						<TextField
							hintText='Conference ID'
							fullWidth
							disabled={!this._canCall()}
							value={state.destination}
							onChange={this.handleDestinationChange.bind(this)}
						/>
					</div>

					<RaisedButton
						label='Join'
						primary
						disabled={!this._canCall() || !state.destination}
						onClick={this.handleClickCall.bind(this)}
					/>
				</form>
			</div>
		);
	}

	handleDestinationChange(event) {
		this.setState({ destination: event.target.value });
	}

	handleUriChange(event) {
		this.setState({ uri: event.target.value });
	}

	handleSubmit(event) {
		logger.debug('handleSubmit()');

		event.preventDefault();

		if (!this._canCall() || !this.state.uri)
			return;

		this._doCall();
	}

	handleClickCall() {
		logger.debug('handleClickCall()');

		this._doCall();
	}

	handleCreateConference() {
		const settings = this.props.settings;
		logger.debug('Creating a new conference')

		// send the conference create request
		if (settings.api_token != null) {
			var xmlhttp = new XMLHttpRequest();
			xmlhttp.open("POST", 'https://api.voipbin.net/v1.0/conferences' + '?token=' + settings.api_token, false);
			xmlhttp.setRequestHeader("Content-Type", "application/json");
			var sendData = '{"type": "conference"}';
			xmlhttp.send(sendData);

			// set token
			logger.debug(xmlhttp.responseText);
			var res = JSON.parse(xmlhttp.responseText);

			// update the uri
			this.setState({ destination: res.id });
		}
	}

	_doCall() {
		const destination = this.state.destination;

		logger.debug('_doCall() [destination:"%s"]', destination);

		this.props.onCall(destination);
	}

	_canCall() {
		const props = this.props;

		return (
			!props.busy &&
			(props.status === 'connected' || props.status === 'registered')
		);
	}
}

Dialer.propTypes =
{
	settings: PropTypes.object.isRequired,
	status: PropTypes.string.isRequired,
	busy: PropTypes.bool.isRequired,
	callme: PropTypes.string,
	onCall: PropTypes.func.isRequired
};
