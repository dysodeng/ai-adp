package errors

import "github.com/dysodeng/ai-adp/internal/domain/shared/errors"

var (
	ErrAppNotFound        = errors.New("APP_NOT_FOUND", "app not found")
	ErrVersionNotFound    = errors.New("VERSION_NOT_FOUND", "app version not found")
	ErrNoPublishedVersion = errors.New("NO_PUBLISHED_VERSION", "no published version for this app")
	ErrDraftAlreadyExists = errors.New("DRAFT_ALREADY_EXISTS", "a draft version already exists for this app")
	ErrCannotEditNonDraft = errors.New("CANNOT_EDIT_NON_DRAFT", "can only edit draft versions")
)
