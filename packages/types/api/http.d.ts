import type { HttpResponse } from '../models/http.js';

export interface HttpApi {
  get(
    url: string,
    params?: Record<string, string>,
    headers?: Record<string, string>,
  ): Promise<HttpResponse>;
  post(url: string, body?: unknown, headers?: Record<string, string>): Promise<HttpResponse>;
}
