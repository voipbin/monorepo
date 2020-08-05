import React from 'react';
import PropTypes from 'prop-types';
import TextField from 'material-ui/TextField';
import FlatButton from 'material-ui/FlatButton';
import SettingsIcon from 'material-ui/svg-icons/action/settings';
import classnames from 'classnames';
import randomString from 'random-string';
import Logger from '../Logger';
import settingsManager from '../settingsManager';
import TransitionAppear from './TransitionAppear';
import Logo from './Logo';
import Settings from './Settings';

const logger = new Logger('Login');

export default class Login extends React.Component
{
	constructor(props)
	{
		super(props);

		this.state =
		{
			settings     : props.settings,
			showSettings : false,
			errors       :
			{
				name : null
			}
		};
	}

	render()
	{
		const state = this.state;
		const settings = state.settings;

		return (
			<TransitionAppear>
				<div data-component='Login'>
					<div className='logo-container'>
						<Logo
							size='big'
						/>
					</div>

					<form action='' onSubmit={this.handleSubmit.bind(this)}>
						{/* <div className='settings-icon-container'>
							<SettingsIcon
								className='icon'
								color='#666'
								hoverColor='#333'
								onClick={this.handleClickSettings.bind(this)}
							/>
						</div> */}

						<div className='form-container'>
							<TextField
								floatingLabelText='Your Name'
								value={settings.display_name || ''}
								errorText={state.errors.name}
								fullWidth
								onChange={this.handleChangeName.bind(this)}
							/>
							<TextField
								floatingLabelText='Username'
								value={settings.api_username || ''}
								errorText={state.errors.name}
								fullWidth
								onChange={this.handleChangeApiUsername.bind(this)}
							/>
							<TextField
								floatingLabelText='Password'
								value={settings.api_password || ''}
								errorText={state.errors.name}
								fullWidth
								type='password'
								onChange={this.handleChangeApiPassword.bind(this)}
							/>
							{/* <FlatButton
								label='Reset'
								primary
								style={{
									display : 'table',
									margin  : '20px auto 0 auto'
								}}
								onClick={this.handleClickReset.bind(this)}
							/> */}
						</div>
					</form>

					<div className='submit-container'>
						<div
							className={classnames('submit-button', { disabled: !this._checkCanPlay() })}
							onClick={this.handleSubmitClick.bind(this)}
						/>
					</div>

					<If condition={state.showSettings}>
						<div className='settings-container'>
							<Settings
								settings={settings}
								onSubmit={this.handleSettingsSubmit.bind(this)}
								onCancel={this.handleSettingsCancel.bind(this)}
							/>
						</div>
					</If>
				</div>
			</TransitionAppear>
		);
	}

	handleChangeName(event)
	{
		const settings = this.state.settings;
		const errors = this.state.errors;
		const name = event.target.value;

		settings['display_name'] = name;
		errors.name = null;

		this.setState({ settings, errors });
	}

	handleChangeApiUsername(event)
	{
		const settings = this.state.settings;
		const errors = this.state.errors;
		const val = event.target.value;

		settings['api_username'] = val;
		errors.name = null;

		this.setState({ settings, errors });
	}

	handleChangeApiPassword(event)
	{
		const settings = this.state.settings;
		const errors = this.state.errors;
		const val = event.target.value;

		settings['api_password'] = val;
		errors.name = null;

		this.setState({ settings, errors });
	}

	handleClickReset()
	{
		logger.debug('handleClickReset()');

		settingsManager.clear();

		this.setState({ settings: settingsManager.get() });
	}

	handleSubmit(event)
	{
		logger.debug('handleSubmit()');

		event.preventDefault();
		this._checkForm();
	}

	handleSubmitClick()
	{
		const settings = this.state.settings;

		logger.debug('handleSubmitClick()');

		// do login if the username and password is not empty
		if (settings.api_username != null && settings.api_password != null) {
			var xmlhttp = new XMLHttpRequest();
			xmlhttp.open("POST", 'https://api.voipbin.net/auth/login', false);
			xmlhttp.setRequestHeader("Content-Type", "application/json");
			var sendData = '{"username": "' + settings.api_username +'", "password": "' + settings.api_password + '"}';
			xmlhttp.send(sendData);

			// set token
			logger.debug(xmlhttp.responseText);
			var res = JSON.parse(xmlhttp.responseText);
			logger.debug('API token: ' + res.token);
			settings.api_token = res.token;
		}

		this._checkForm();
	}

	handleClickSettings()
	{
		logger.debug('handleClickSettings()');

		this.setState({ showSettings: true });
	}

	handleSettingsSubmit(settings)
	{
		logger.debug('handleSettingsSubmit()');

		this.setState({ settings, showSettings: false });
	}

	handleSettingsCancel()
	{
		logger.debug('handleSettingsCancel()');

		this.setState({ showSettings: false });
	}

	_checkCanPlay()
	{
		const state = this.state;

		if (state.settings.display_name && !state.showSettings)
			return true;
		else
			return false;
	}

	_checkForm()
	{
		logger.debug('_checkForm()');

		const state = this.state;
		const settings = state.settings;
		const errors = state.errors;
		let ok = true;

		// Check name
		{
			if (settings.display_name.length < 3)
			{
				ok = false;
				errors.name = 'Name too short';
			}
		}

		if (!ok)
		{
			this.setState({ errors });

			return;
		}

		// If no SIP URI is set, set a random one
		if (!settings.uri)
		{
			const username =
				`${settings.display_name.toLowerCase().replace(/[^a-zA-Z0-9]/g, '')}_${randomString({ length: 6 }).toLowerCase()}`;
			const domain = settingsManager.getDefaultDomain();

			settings.uri = `sip:${username}@${domain}`;
		}

		// Fire event
		this.props.onLogin(settings);
	}
}

Login.propTypes =
{
	settings : PropTypes.object.isRequired,
	onLogin  : PropTypes.func.isRequired
};
