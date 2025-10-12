package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/ntentasd/nostradamus-api/pkg/types"
)

var ErrFieldNotFound = errors.New("field not found")

type FieldAlreadyExistsError struct {
	FieldName string
}

func (e *FieldAlreadyExistsError) Error() string {
	return fmt.Sprintf("field '%s' already exists", e.FieldName)
}

func (e *FieldAlreadyExistsError) Is(target error) bool {
	_, ok := target.(*FieldAlreadyExistsError)
	return ok
}

func (db *DB) GetFieldsByUserID(userID uuid.UUID) ([]types.Field, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	query := db.Meta.Query(`
SELECT field_id, field_name
FROM fields
WHERE user_id = ?
`, gocql.UUID(userID)).WithContext(ctx)

	var results []types.Field
	iter := query.Iter()

	var fieldID uuid.UUID
	var fieldName string

	for iter.Scan(&fieldID, &fieldName) {
		results = append(results, types.Field{
			UserID:    &userID,
			FieldID:   fieldID,
			FieldName: fieldName,
		})
	}

	if err := iter.Close(); err != nil {
		return nil, err
	}

	return results, nil
}

func (db *DB) GetFieldByID(fieldID uuid.UUID) (*types.Field, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	var userID gocql.UUID
	var fieldName string

	err := db.Meta.Query(`
SELECT user_id, field_name
FROM fields
WHERE field_id = ?
ALLOW FILTERING
`, gocql.UUID(fieldID)).WithContext(ctx).Scan(&userID, &fieldName)
	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &types.Field{
		UserID:    (*uuid.UUID)(&userID),
		FieldID:   fieldID,
		FieldName: fieldName,
	}, nil
}

func (db *DB) RegisterField(userID uuid.UUID, fieldName string) (*types.Field, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	checkQuery := db.Meta.Query(`
SELECT field_id FROM fields
WHERE user_id = ?
`, gocql.UUID(userID)).WithContext(ctx)

	iter := checkQuery.Iter()
	var existingID gocql.UUID
	for iter.Scan(&existingID) {
		var existingName string
		subQuery := db.Meta.Query(`
SELECT field_name FROM fields
WHERE user_id = ? AND field_id = ?
`, gocql.UUID(userID), existingID).WithContext(ctx)
		if err := subQuery.Scan(&existingName); err == nil && existingName == fieldName {
			return nil, &FieldAlreadyExistsError{
				FieldName: fieldName,
			}
		}
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}

	fieldID := uuid.New()
	insertQuery := db.Meta.Query(`
INSERT INTO fields (user_id, field_id, field_name)
VALUES (?, ?, ?)
`, gocql.UUID(userID), gocql.UUID(fieldID), fieldName).WithContext(ctx)

	if err := insertQuery.Exec(); err != nil {
		return nil, err
	}

	return &types.Field{
		UserID:    &userID,
		FieldID:   fieldID,
		FieldName: fieldName,
	}, nil
}
