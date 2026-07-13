package model

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

const (
	VolcAssetAuthorizationEventConsentAccepted = "ConsentAccepted"
	VolcAssetAuthorizationEventAuthorized      = "Authorized"
	VolcAssetAuthorizationEventRevoked         = "Revoked"
)

var ErrVolcAssetConsentRequired = errors.New("Seedance actor consent is required")

type VolcAssetAuthorizationEvent struct {
	Id             int    `json:"id" gorm:"primaryKey"`
	UserId         int    `json:"user_id" gorm:"index;not null"`
	ActorId        int    `json:"actor_id" gorm:"index;not null"`
	SessionId      int    `json:"session_id,omitempty" gorm:"index"`
	EventType      string `json:"event_type" gorm:"type:varchar(32);index;not null"`
	DurationDays   *int   `json:"duration_days,omitempty"`
	ConsentVersion string `json:"consent_version,omitempty" gorm:"type:varchar(32)"`
	CreatedAt      int64  `json:"created_at" gorm:"type:bigint;not null"`
}

type VolcAssetAuthorizationInvitation struct {
	Session VolcAssetAuthorizationSession
	Actor   VolcAssetActor
}

func validateVolcAssetAuthorizationDigest(digest string) error {
	return validateVolcAssetAuthorizationInput(1, digest)
}

func StartVolcAssetActorAuthorization(
	userId, actorId int,
	tokenHash, invitationHash, revocationHash string,
	durationDays *int,
	consentVersion string,
	expiresAt, now int64,
) (*VolcAssetAuthorizationSession, error) {
	if userId <= 0 || actorId <= 0 || expiresAt <= now || consentVersion == "" {
		return nil, fmt.Errorf("invalid Seedance actor authorization session")
	}
	for _, digest := range []string{tokenHash, invitationHash, revocationHash} {
		if err := validateVolcAssetAuthorizationDigest(digest); err != nil {
			return nil, err
		}
	}
	if err := validateVolcAssetActorDuration(durationDays); err != nil {
		return nil, err
	}
	actor, err := GetVolcAssetActorForUser(userId, actorId)
	if err != nil {
		return nil, err
	}
	if !equalOptionalInt(actor.AuthorizationDurationDays, durationDays) {
		return nil, ErrVolcAssetInvalidDuration
	}

	session := &VolcAssetAuthorizationSession{
		UserId:              userId,
		ActorId:             actorId,
		TokenHash:           tokenHash,
		InvitationTokenHash: invitationHash,
		RevocationTokenHash: revocationHash,
		DurationDays:        durationDays,
		ConsentVersion:      consentVersion,
		Status:              VolcAssetAuthorizationPending,
		ExpiresAt:           expiresAt,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&VolcAssetAuthorizationSession{}).
			Where("user_id = ? AND actor_id = ? AND status = ?", userId, actorId, VolcAssetAuthorizationPending).
			Updates(map[string]any{"status": VolcAssetAuthorizationExpired, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Create(session).Error
	})
	if err != nil {
		return nil, err
	}
	return session, nil
}

func FindPendingVolcAssetActorAuthorization(userId, actorId int, tokenHash string, now int64) (*VolcAssetAuthorizationSession, error) {
	if userId <= 0 || actorId <= 0 {
		return nil, ErrVolcAssetAuthorizationNotFound
	}
	if err := validateVolcAssetAuthorizationDigest(tokenHash); err != nil {
		return nil, err
	}
	var session VolcAssetAuthorizationSession
	err := DB.Where("user_id = ? AND actor_id = ? AND token_hash = ?", userId, actorId, tokenHash).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVolcAssetAuthorizationNotFound
	}
	if err != nil {
		return nil, err
	}
	if session.Status != VolcAssetAuthorizationPending {
		return nil, ErrVolcAssetAuthorizationNotFound
	}
	if session.ExpiresAt <= now {
		if err := DB.Model(&session).Where("status = ?", VolcAssetAuthorizationPending).
			Updates(map[string]any{"status": VolcAssetAuthorizationExpired, "updated_at": now}).Error; err != nil {
			return nil, err
		}
		return nil, ErrVolcAssetAuthorizationExpired
	}
	return &session, nil
}

func ExpireVolcAssetActorAuthorization(userId, actorId int, tokenHash string, now int64) error {
	if userId <= 0 || actorId <= 0 {
		return ErrVolcAssetAuthorizationNotFound
	}
	if err := validateVolcAssetAuthorizationDigest(tokenHash); err != nil {
		return err
	}
	result := DB.Model(&VolcAssetAuthorizationSession{}).
		Where("user_id = ? AND actor_id = ? AND token_hash = ? AND status = ?", userId, actorId, tokenHash, VolcAssetAuthorizationPending).
		Updates(map[string]any{"status": VolcAssetAuthorizationExpired, "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrVolcAssetAuthorizationNotFound
	}
	return nil
}

func GetVolcAssetAuthorizationInvitation(invitationHash string, now int64) (*VolcAssetAuthorizationInvitation, error) {
	if err := validateVolcAssetAuthorizationDigest(invitationHash); err != nil {
		return nil, ErrVolcAssetAuthorizationNotFound
	}
	var session VolcAssetAuthorizationSession
	err := DB.Where("invitation_token_hash = ? AND status = ?", invitationHash, VolcAssetAuthorizationPending).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVolcAssetAuthorizationNotFound
	}
	if err != nil {
		return nil, err
	}
	if session.ExpiresAt <= now {
		_ = DB.Model(&session).Updates(map[string]any{"status": VolcAssetAuthorizationExpired, "updated_at": now}).Error
		return nil, ErrVolcAssetAuthorizationExpired
	}
	actor, err := GetVolcAssetActorForUser(session.UserId, session.ActorId)
	if err != nil {
		return nil, err
	}
	actor.GroupId = ""
	return &VolcAssetAuthorizationInvitation{Session: session, Actor: *actor}, nil
}

func AcceptVolcAssetAuthorizationConsent(invitationHash string, now int64) (*VolcAssetAuthorizationSession, error) {
	invitation, err := GetVolcAssetAuthorizationInvitation(invitationHash, now)
	if err != nil {
		return nil, err
	}
	if invitation.Session.ConsentAcceptedAt != nil {
		return &invitation.Session, nil
	}
	var accepted VolcAssetAuthorizationSession
	err = DB.Transaction(func(tx *gorm.DB) error {
		acceptedAt := now
		result := tx.Model(&VolcAssetAuthorizationSession{}).
			Where("id = ? AND status = ? AND consent_accepted_at IS NULL AND expires_at > ?", invitation.Session.Id, VolcAssetAuthorizationPending, now).
			Updates(map[string]any{"consent_accepted_at": acceptedAt, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrVolcAssetAuthorizationNotFound
		}
		if err := tx.First(&accepted, invitation.Session.Id).Error; err != nil {
			return err
		}
		return tx.Create(&VolcAssetAuthorizationEvent{
			UserId: accepted.UserId, ActorId: accepted.ActorId, SessionId: accepted.Id,
			EventType: VolcAssetAuthorizationEventConsentAccepted, DurationDays: accepted.DurationDays,
			ConsentVersion: accepted.ConsentVersion, CreatedAt: now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return &accepted, nil
}

func CompleteVolcAssetActorAuthorization(userId, actorId int, tokenHash, groupId string, now int64) error {
	if err := validateVolcAssetAuthorizationDigest(tokenHash); err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var session VolcAssetAuthorizationSession
		err := tx.Where("user_id = ? AND actor_id = ? AND token_hash = ?", userId, actorId, tokenHash).First(&session).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVolcAssetAuthorizationNotFound
		}
		if err != nil {
			return err
		}
		if session.Status == VolcAssetAuthorizationCompleted {
			return nil
		}
		if session.Status != VolcAssetAuthorizationPending {
			return ErrVolcAssetAuthorizationNotFound
		}
		if session.ExpiresAt <= now {
			if err := tx.Model(&session).Updates(map[string]any{"status": VolcAssetAuthorizationExpired, "updated_at": now}).Error; err != nil {
				return err
			}
			return ErrVolcAssetAuthorizationExpired
		}
		if session.ConsentAcceptedAt == nil {
			return ErrVolcAssetConsentRequired
		}
		if err := tx.Model(&VolcAssetActor{}).
			Where("user_id = ? AND id = ?", userId, actorId).
			Update("authorization_duration_days", session.DurationDays).Error; err != nil {
			return err
		}
		if err := activateVolcAssetActor(tx, userId, actorId, groupId, session.ConsentVersion, now); err != nil {
			return err
		}
		if err := tx.Model(&session).Updates(map[string]any{"status": VolcAssetAuthorizationCompleted, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Create(&VolcAssetAuthorizationEvent{
			UserId: userId, ActorId: actorId, SessionId: session.Id,
			EventType: VolcAssetAuthorizationEventAuthorized, DurationDays: session.DurationDays,
			ConsentVersion: session.ConsentVersion, CreatedAt: now,
		}).Error
	})
}

func RevokeVolcAssetActorByToken(revocationHash string, now int64) error {
	if err := validateVolcAssetAuthorizationDigest(revocationHash); err != nil {
		return ErrVolcAssetAuthorizationNotFound
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var session VolcAssetAuthorizationSession
		err := tx.Where("revocation_token_hash = ? AND status = ?", revocationHash, VolcAssetAuthorizationCompleted).First(&session).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVolcAssetAuthorizationNotFound
		}
		if err != nil {
			return err
		}
		var actor VolcAssetActor
		if err := tx.Where("user_id = ? AND id = ?", session.UserId, session.ActorId).First(&actor).Error; err != nil {
			return err
		}
		if actor.Status == VolcAssetActorRevoked {
			return nil
		}
		if err := tx.Model(&actor).Updates(map[string]any{"status": VolcAssetActorRevoked, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Create(&VolcAssetAuthorizationEvent{
			UserId: session.UserId, ActorId: session.ActorId, SessionId: session.Id,
			EventType: VolcAssetAuthorizationEventRevoked, DurationDays: session.DurationDays,
			ConsentVersion: session.ConsentVersion, CreatedAt: now,
		}).Error
	})
}

func equalOptionalInt(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
