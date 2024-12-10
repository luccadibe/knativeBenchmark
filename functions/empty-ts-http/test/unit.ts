'use strict';

import test from 'tape';
import { expectType } from 'tsd';
import { Context, HTTPFunction } from 'faas-js-runtime';
import { handle } from '../build/index.js';

test('Unit: handles a valid request', async (t) => {
  t.plan(2);

  const result = await handle({} as Context, '');
  t.ok(result);
  t.equal(result.body, '');
  t.end();
});

expectType<HTTPFunction>(handle);
