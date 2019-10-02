/*
Copyright 2019 The Tekton Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import React from 'react';
import { storiesOf } from '@storybook/react';
import StoryRouter from 'storybook-react-router';

import RunDropdown from './RunDropdown';

storiesOf('RunDropdown', module)
  .addDecorator(StoryRouter())
  .add('default', () => (
    <RunDropdown
      items={[
        { actionText: 'Rebuild', action: () => {} },
        {
          actionText: 'Delete',
          action: () => {},
          modalProperties: {
            heading: 'Modal Heading',
            primaryButtonText: 'primary text',
            secondaryButtonText: 'secondary text',
            body: () => 'modal body'
          }
        },
        {
          actionText: 'disabled option',
          disabled: true,
          action: () => {},
          modalProperties: {
            heading: 'Modal Heading',
            primaryButtonText: 'primary text',
            secondaryButtonText: 'secondary text',
            body: () => 'modal body'
          }
        }
      ]}
    />
  ));
