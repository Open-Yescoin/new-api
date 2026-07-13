/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { API } from '../helpers/api';

export const seedanceAssetsApi = {
  listActors: () =>
    API.get('/api/seedance/actors', {
      disableDuplicate: true,
      skipErrorHandler: true,
    }),
  createActor: (payload) =>
    API.post('/api/seedance/actors', payload, { skipErrorHandler: true }),
  renameActor: (actorId, displayName) =>
    API.patch(
      `/api/seedance/actors/${encodeURIComponent(actorId)}`,
      { display_name: displayName },
      { skipErrorHandler: true },
    ),
  deleteActor: (actorId) =>
    API.delete(`/api/seedance/actors/${encodeURIComponent(actorId)}`, {
      skipErrorHandler: true,
    }),
  actorAuthorizationStatus: (actorId) =>
    API.get(
      `/api/seedance/actors/${encodeURIComponent(actorId)}/authorization`,
      { disableDuplicate: true, skipErrorHandler: true },
    ),
  createActorAuthorizationSession: (actorId) =>
    API.post(
      `/api/seedance/actors/${encodeURIComponent(actorId)}/authorization/session`,
      undefined,
      { skipErrorHandler: true },
    ),
  actorAuthorizationResult: (actorId, bytedToken) =>
    API.post(
      `/api/seedance/actors/${encodeURIComponent(actorId)}/authorization/result`,
      { byted_token: bytedToken },
      { skipErrorHandler: true },
    ),
  revokeActor: (actorId) =>
    API.post(
      `/api/seedance/actors/${encodeURIComponent(actorId)}/authorization/revoke`,
      undefined,
      { skipErrorHandler: true },
    ),
  listActorAssets: (actorId, params) =>
    API.get(`/api/seedance/actors/${encodeURIComponent(actorId)}/assets`, {
      params,
      skipErrorHandler: true,
    }),
  createActorAsset: (actorId, payload) =>
    API.post(
      `/api/seedance/actors/${encodeURIComponent(actorId)}/assets`,
      payload,
      { skipErrorHandler: true },
    ),
  updateActorAsset: (actorId, assetId, name) =>
    API.patch(
      `/api/seedance/actors/${encodeURIComponent(actorId)}/assets/${encodeURIComponent(assetId)}`,
      { name },
      { skipErrorHandler: true },
    ),
  removeActorAsset: (actorId, assetId) =>
    API.delete(
      `/api/seedance/actors/${encodeURIComponent(actorId)}/assets/${encodeURIComponent(assetId)}`,
      { skipErrorHandler: true },
    ),
  publicInvitationDetails: (token) =>
    API.post(
      '/api/seedance/public/authorization/details',
      { token },
      { skipErrorHandler: true },
    ),
  publicAcceptConsent: (token) =>
    API.post(
      '/api/seedance/public/authorization/consent',
      { token },
      { skipErrorHandler: true },
    ),
  publicRevoke: (token) =>
    API.post(
      '/api/seedance/public/authorization/revoke',
      { token },
      { skipErrorHandler: true },
    ),
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
    API.get('/api/seedance/assets', {
      params,
      skipErrorHandler: true,
    }),
  create: (payload) =>
    API.post('/api/seedance/assets', payload, { skipErrorHandler: true }),
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
