package domain

import (
	"errors"
	"fmt"
)

var (
	// Base errors
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrInternal      = errors.New("internal error")
	
	// Track specific errors
	ErrTrackCorrupted    = errors.New("track file is corrupted")
	ErrTrackUnsupported  = errors.New("track format is not supported")
	ErrTrackMetadataMissing = errors.New("track metadata is missing")
	
	// Playlist specific errors  
	ErrPlaylistLocked    = errors.New("playlist is locked for editing")
	ErrPlaylistCorrupted = errors.New("playlist file is corrupted")
	ErrCircularReference = errors.New("circular reference detected")
	
	// Library specific errors
	ErrLibraryScanning   = errors.New("library is currently scanning")
	ErrLibraryCorrupted  = errors.New("library database is corrupted")
	ErrPathNotAccessible = errors.New("path is not accessible")
	
	// Audio engine errors
	ErrAudioDeviceNotFound = errors.New("audio device not found")
	ErrAudioFormatMismatch = errors.New("audio format mismatch")
	ErrAudioBufferOverrun  = errors.New("audio buffer overrun")
	ErrAudioBufferUnderrun = errors.New("audio buffer underrun")
	
	// Network errors
	ErrNetworkTimeout     = errors.New("network operation timed out")
	ErrNetworkUnavailable = errors.New("network is unavailable")
	ErrInvalidCredentials = errors.New("invalid network credentials")
	
	// File system errors
	ErrFileNotFound      = errors.New("file not found")
	ErrFileAccessDenied  = errors.New("file access denied")
	ErrFileReadOnly      = errors.New("file is read-only")
	ErrDiskFull          = errors.New("disk is full")
)

type DomainError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Err     error  `json:"-"`
}

func (e *DomainError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

func NewDomainError(code string, message string, err error) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func NewDomainErrorWithDetails(code string, message string, details string, err error) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Details: details,
		Err:     err,
	}
}

// Error codes for consistent error handling
const (
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeAlreadyExists     = "ALREADY_EXISTS"
	ErrCodeInvalidInput      = "INVALID_INPUT"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeInternal          = "INTERNAL"
	ErrCodeTrackCorrupted    = "TRACK_CORRUPTED"
	ErrCodeTrackUnsupported  = "TRACK_UNSUPPORTED"
	ErrCodePlaylistLocked    = "PLAYLIST_LOCKED"
	ErrCodeLibraryScanning   = "LIBRARY_SCANNING"
	ErrCodeAudioDevice       = "AUDIO_DEVICE"
	ErrCodeNetwork           = "NETWORK"
	ErrCodeFileSystem        = "FILE_SYSTEM"
)

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, ErrTrackNotFound) || errors.Is(err, ErrFileNotFound)
}

func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists) || errors.Is(err, ErrDuplicateLibraryPath)
}

func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrInvalidTrack) || 
	       errors.Is(err, ErrInvalidPlaylist) || errors.Is(err, ErrInvalidPosition)
}

func IsAudioError(err error) bool {
	return errors.Is(err, ErrAudioDeviceNotFound) || errors.Is(err, ErrAudioFormatMismatch) ||
	       errors.Is(err, ErrAudioBufferOverrun) || errors.Is(err, ErrAudioBufferUnderrun)
}

func IsNetworkError(err error) bool {
	return errors.Is(err, ErrNetworkTimeout) || errors.Is(err, ErrNetworkUnavailable) ||
	       errors.Is(err, ErrInvalidCredentials)
}

func IsFileSystemError(err error) bool {
	return errors.Is(err, ErrFileNotFound) || errors.Is(err, ErrFileAccessDenied) ||
	       errors.Is(err, ErrFileReadOnly) || errors.Is(err, ErrDiskFull)
}