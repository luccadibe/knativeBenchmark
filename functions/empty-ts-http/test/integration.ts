'use strict';
import { start, InvokerOptions } from 'faas-js-runtime';
import request from 'supertest';

import * as func from '../build';
import test, { Test } from 'tape';

const errHandler = (t: Test) => (err: Error) => {
  t.error(err);
  t.end();
};

test('Integration: handles a valid request', (t) => {
  start(func.handle, {} as InvokerOptions).then((server) => {
    t.plan(3);
    request(server)
      .post('/')
      .send('')
      .expect(200)
      .expect('Content-Type', /text\/plain/)
      .end((err, result) => {
        t.error(err, 'No error');
        t.ok(result);
        t.equal(result.text, '');
        t.end();
        server.close();
      });
  }, errHandler(t));
});
