import { Context, StructuredReturn } from 'faas-js-runtime';

const handle = async (_context: Context, _body: string): Promise<StructuredReturn> => {
  return {
    body: '',
    headers: {
      'content-type': 'text/plain'
    }
  };
};

export { handle };
