import { API } from '../helpers/api';

export const seedanceAssetsApi = {
  authorizationStatus: () =>
    API.get('/api/seedance/assets/authorization', {
      disableDuplicate: true,
    }),
  createAuthorizationSession: () =>
    API.post('/api/seedance/assets/authorization/session'),
  authorizationResult: (bytedToken) =>
    API.post('/api/seedance/assets/authorization/result', {
      byted_token: bytedToken,
    }),
  list: (params) => API.get('/api/seedance/assets/', { params }),
  create: (payload) => API.post('/api/seedance/assets/', payload),
  get: (id) =>
    API.get(`/api/seedance/assets/${encodeURIComponent(id)}`, {
      disableDuplicate: true,
    }),
  update: (id, name) =>
    API.patch(`/api/seedance/assets/${encodeURIComponent(id)}`, { name }),
  remove: (id) => API.delete(`/api/seedance/assets/${encodeURIComponent(id)}`),
};
