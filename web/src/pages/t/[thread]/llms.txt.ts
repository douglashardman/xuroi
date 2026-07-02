export const prerender = false;

import type { APIRoute } from 'astro';
import { fetchThreadLLMText } from '../../../lib/thread-export';

export const GET: APIRoute = async ({ params }) => {
  return fetchThreadLLMText(params.thread ?? '');
};