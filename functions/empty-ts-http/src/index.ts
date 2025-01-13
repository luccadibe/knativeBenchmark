import { Context, StructuredReturn } from 'faas-js-runtime';

let isCold = true;

const handle = async (_context: Context, _body: string): Promise<StructuredReturn> => {
  const response = isCold.toString();
  isCold = false;
  return {
    body: response,
    headers: {
      'content-type': 'text/plain'
    }
  };
};

export { handle };