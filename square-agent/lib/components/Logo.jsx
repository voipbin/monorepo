import React from 'react';
import PropTypes from 'prop-types';

export default class Logo extends React.Component
{
	constructor(props)
	{
		super(props);

		this.state = {};
	}

	render()
	{
		const size = this.props.size || 'big';

		return (
			<div data-component='Logo' className={size}>
				<h1>
					<span className='colorized'>AGENT</span>.V<span className='small'>oIP</span>BIN.net
				</h1>
			</div>
		);
	}
}

Logo.propTypes =
{
	size : PropTypes.string
};
