import type { Catalog } from './types';

/** State the node components read without prop drilling. */
export const editor = $state({
  catalog: {} as Catalog,
  output: '',
});
