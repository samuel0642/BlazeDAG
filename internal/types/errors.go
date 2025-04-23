package types

import "errors"

// Error definitions
var (
	ErrInvalidBlock        = errors.New("invalid block")
	ErrInvalidTransaction  = errors.New("invalid transaction")
	ErrInvalidVote        = errors.New("invalid vote")
	ErrInvalidCertificate = errors.New("invalid certificate")
	ErrAccountNotFound    = errors.New("account not found")
	ErrAccountExists      = errors.New("account already exists")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrInvalidNonce       = errors.New("invalid nonce")
	ErrValidatorNotFound  = errors.New("validator not found")
	ErrInvalidBalance     = errors.New("invalid balance")
) 