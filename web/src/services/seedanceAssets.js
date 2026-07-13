import { API } from '../helpers/api';

export const seedanceAssetsApi = {
  authorizationStatus: () =>
    API.get('/api/seedance/assets/authorization', {
      disableDuplicate: true,
      skipErrorHandler: true,
    }),
  createAuthorizationSession: () =>
    API.post('/api/seedance/assets/authorization/session', undefined, {
      skipErrorHandler: true,
    }),
  authorizationResult: (bytedToken) =>
    API.post(
      '/api/seedance/assets/authorization/result',
      { byted_token: bytedToken },
      { skipErrorHandler: true },
    ),
  list: (params) =>
    API.get('/api/seedance/assets/', {
      params,
      skipErrorHandler: true,
    }),
  create: (payload) =>
    API.post('/api/seedance/assets/', payload, { skipErrorHandler: true }),
  get: (id) =>
    API.get(`/api/seedance/assets/${encodeURIComponent(id)}`, {
      disableDuplicate: true,
      skipErrorHandler: true,
    }),
  update: (id, name) =>
    API.patch(
      `/api/seedance/assets/${encodeURIComponent(id)}`,
      { name },
      { skipErrorHandler: true },
    ),
  remove: (id) =>
    API.delete(`/api/seedance/assets/${encodeURIComponent(id)}`, {
      skipErrorHandler: true,
    }),
};
