// Copyright © 2019 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import React from 'react'
import classnames from 'classnames'
import bind from 'autobind-decorator'

import Select from '../../../components/select'
import Input from '../../../components/input'
import { unit as unitRegexp } from '../../lib/regexp'
import PropTypes from '../../../lib/prop-types'
import style from './duration-input.styl'

class DurationInput extends React.PureComponent {
  static propTypes = {
    className: PropTypes.string,
    decode: PropTypes.func,
    encode: PropTypes.func,
    name: PropTypes.string.isRequired,
    onChange: PropTypes.func.isRequired,
    units: PropTypes.arrayOf(
      PropTypes.shape({
        label: PropTypes.message,
        value: PropTypes.string,
      }),
    ).isRequired,
    value: PropTypes.string,
  }

  static defaultProps = {
    className: undefined,
    encode: (duration, unit) => (duration ? `${duration}${unit}` : ''),
    decode: value => {
      const duration = value.split(unitRegexp)[0]
      const unit = value.split(duration)[1]
      return {
        duration: Number(duration),
        unit,
      }
    },
    value: undefined,
  }

  constructor(props) {
    super(props)
    const { value, decode } = this.props

    const initialDelay = decode(value)
    this.state = {
      duration: initialDelay.duration,
      unit: initialDelay.unit,
    }
  }

  @bind
  async handleChange(value) {
    const { onChange, encode } = this.props
    const { unit } = this.state

    this.setState({ duration: value })
    // if duration is empty, propagate empty value to form
    onChange(encode(value, unit))
  }

  @bind
  async handleUnitChange(value) {
    const { onChange, encode } = this.props
    const { duration } = this.state
    await this.setState({ unit: value })
    // if duration is empty, propagate empty value to form
    onChange(encode(duration, value))
  }

  render() {
    const { className, name, units } = this.props
    const { duration, unit } = this.state
    const selectTimeUnitComponent = (
      <Select
        className={style.select}
        name={`${name}-select`}
        options={units}
        onChange={this.handleUnitChange}
        value={unit}
      />
    )

    return (
      <React.Fragment>
        <div className={classnames(className, style.container)}>
          <Input
            className={style.number}
            ref={this.inputRef}
            type="number"
            step="any"
            name={name}
            onBlur={this.handleBlur}
            value={duration}
            onChange={this.handleChange}
          />
          {selectTimeUnitComponent}
        </div>
      </React.Fragment>
    )
  }
}

export default DurationInput
