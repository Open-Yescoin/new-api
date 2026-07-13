package doubao

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/model"
)

type ActorService struct {
	now func() time.Time
}

type Actor struct {
	Id                        int    `json:"id"`
	DisplayName               string `json:"display_name"`
	Status                    string `json:"status"`
	AuthorizationDurationDays *int   `json:"authorization_duration_days,omitempty"`
	AuthorizedAt              *int64 `json:"authorized_at,omitempty"`
	AuthorizationExpiresAt    *int64 `json:"authorization_expires_at,omitempty"`
	ConsentVersion            string `json:"consent_version,omitempty"`
	IsDefault                 bool   `json:"is_default"`
	CreatedAt                 int64  `json:"created_at"`
	UpdatedAt                 int64  `json:"updated_at"`
}

func NewActorService(now func() time.Time) *ActorService {
	if now == nil {
		now = time.Now
	}
	return &ActorService{now: now}
}

func (s *ActorService) Create(userId int, displayName string, durationDays *int) (*Actor, error) {
	actor, err := model.CreateVolcAssetActor(userId, displayName, durationDays, s.now().Unix())
	if err != nil {
		return nil, err
	}
	result := newActor(actor)
	return &result, nil
}

func (s *ActorService) List(userId int) ([]Actor, error) {
	actors, err := model.ListVolcAssetActors(userId)
	if err != nil {
		return nil, err
	}
	now := s.now().Unix()
	result := make([]Actor, 0, len(actors))
	for i := range actors {
		if actors[i].AuthorizationExpiresAt != nil && *actors[i].AuthorizationExpiresAt <= now && actors[i].Status != model.VolcAssetActorExpired {
			_, guardErr := model.RequireVolcAssetActorActiveAt(userId, actors[i].Id, now)
			if guardErr != nil && !errors.Is(guardErr, model.ErrVolcAssetActorExpired) {
				return nil, guardErr
			}
			refreshed, getErr := model.GetVolcAssetActorForUser(userId, actors[i].Id)
			if getErr != nil {
				return nil, getErr
			}
			actors[i] = *refreshed
		}
		result = append(result, newActor(&actors[i]))
	}
	return result, nil
}

func (s *ActorService) Get(userId, actorId int) (*Actor, error) {
	actor, err := model.GetVolcAssetActorForUser(userId, actorId)
	if err != nil {
		return nil, err
	}
	if actor.AuthorizationExpiresAt != nil && *actor.AuthorizationExpiresAt <= s.now().Unix() && actor.Status != model.VolcAssetActorExpired {
		_, guardErr := model.RequireVolcAssetActorActiveAt(userId, actorId, s.now().Unix())
		if guardErr != nil && !errors.Is(guardErr, model.ErrVolcAssetActorExpired) {
			return nil, guardErr
		}
		actor, err = model.GetVolcAssetActorForUser(userId, actorId)
		if err != nil {
			return nil, err
		}
	}
	result := newActor(actor)
	return &result, nil
}

func (s *ActorService) Rename(userId, actorId int, displayName string) error {
	return model.RenameVolcAssetActor(userId, actorId, displayName, s.now().Unix())
}

func (s *ActorService) Delete(userId, actorId int) error {
	return model.DeleteVolcAssetActor(userId, actorId)
}

func (s *ActorService) Revoke(userId, actorId int) error {
	return model.RevokeVolcAssetActor(userId, actorId, s.now().Unix())
}

func newActor(actor *model.VolcAssetActor) Actor {
	return Actor{
		Id: actor.Id, DisplayName: actor.DisplayName, Status: actor.Status,
		AuthorizationDurationDays: actor.AuthorizationDurationDays,
		AuthorizedAt:              actor.AuthorizedAt, AuthorizationExpiresAt: actor.AuthorizationExpiresAt,
		ConsentVersion: actor.ConsentVersion, IsDefault: actor.IsDefault,
		CreatedAt: actor.CreatedAt, UpdatedAt: actor.UpdatedAt,
	}
}
