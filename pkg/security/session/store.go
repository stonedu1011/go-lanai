package session

import (
	"bytes"
	"context"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/redis"
	"encoding/gob"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const sessionValueField = "values"
const sessionOptionField = "options"
const sessionLastAccessedField = "lastAccessed"

type Store interface {
	// Get should return a cached session.
	Get(id string, name string) (*Session, error)

	// New should create and return a new session.
	New(name string) (*Session, error)

	// Save should persist session to the underlying store implementation.
	Save(s *Session) error

	//Delete the session from store. It will also update the http response header to clear the corresponding cookie
	Delete(s *Session) error

	Options() *Options

	FindByPrincipalName(principal string) ([]*Session, error)

	ChangeId(s *Session) error
}

/**
	Session is implemented as a HSET in redis.
	Session is expired using Redis TTL. The TTL is slightly longer than the expiration time so that when the session is
	expired, we can still get the session details (if necessary).

	Currently we don't have a need for "on session expired" event. If we do have a need for this in the future,
	we can use https://redis.io/topics/notifications and listen on the TTL event. Once caveat of using the redis
	notification is that it may not generate a event until the key is being accessed. So if we want to have deterministic
	behaviour on when the event is fired, we would need to implement a scheduler ourselves that access these keys that
	are expired which will force redis to generate the event.
 */
type RedisStore struct {
	options    *Options
	connection *redis.Connection
}

func NewRedisStore(connection *redis.Connection, options ...func(*Options)) *RedisStore {
	gob.Register(time.Time{})

	//defaults
	o := &Options{
		Path:   "/",
		HttpOnly: true,
		SameSite: http.SameSiteDefaultMode,
		IdleTimeout: 900*time.Second,
		AbsoluteTimeout: 1800*time.Second,
	}

	for _, opt := range options {
		opt(o)
	}

	s := &RedisStore{
		options: o,
		connection: connection,
	}
	return s
}

func (s *RedisStore) Options() *Options {
	return s.options
}

func (s *RedisStore) Get(id string, name string) (*Session, error) {
	if id != "" {
		session, err := s.load(id, name)

		if err != nil {
			return nil, err
		}

		if session == nil {
			return s.New(name)
		} else {
			return session, nil
		}
	} else {
		return s.New(name)
	}
}

// New will create a new session.
func (s *RedisStore) New(name string) (*Session, error) {
	session := NewSession(s, name)

	session.lastAccessed = time.Now()
	session.values[createdTimeKey] = time.Now()

	session.id = uuid.New().String()

	return session, nil
}

// Save adds a single session to the persistence layer
func (s *RedisStore) Save(session *Session) error {
	if session.id == "" {
		return errors.New("session id is empty")
	}

	session.lastAccessed = time.Now()
	err := s.save(session)
	if err == nil {
		session.dirty = false
	}
	return err
}

func (s *RedisStore) Delete(session *Session) error {
	cmd := s.connection.Del(context.Background(), session.id)
	return cmd.Err()
}

func (s *RedisStore) FindByPrincipalName(principal string) ([]*Session, error) {
	//TODO:
	return []*Session{}, nil
}

func (s *RedisStore) ChangeId(session *Session) error {
	newId := uuid.New().String()
	cmd := s.connection.Rename(context.Background(), getKey(session.Name(), session.id), getKey(session.Name(), newId))
	err := cmd.Err()
	if err != nil {
		return err
	}
	session.id = newId
	return nil
}

func (s *RedisStore) load(id string, name string) (*Session, error) {
	key := getKey(name, id)

	cmd := s.connection.HGetAll(context.Background(), key)

	result, err := cmd.Result()

	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	session := NewSession(s, name)
	session.id = id

	for k, v := range result {
		if k == sessionValueField {
			err = Deserialize(strings.NewReader(v), &session.values)
		} else if k == sessionOptionField {
			err = Deserialize(strings.NewReader(v), &session.options)
		} else if k == sessionLastAccessedField {
			timeStamp, e := strconv.ParseInt(v, 10, 0)
			session.lastAccessed = time.Unix(timeStamp, 0)
			err = e
		}

		if err != nil {
			return nil, err
		}
	}
	session.isNew = false

	if session.isExpired() {
		return nil, nil
	} else {
		return session, nil
	}
}

func (s *RedisStore) save(session *Session) error {
	key := getKey(session.Name(), session.id)
	var args []interface{}

	if session.IsDirty() || session.isNew {
		if values, err := Serialize(session.values); err == nil {
			args = append(args, sessionValueField, values)
		} else {
			return err
		}
	}

	if session.isNew {
		if options, err := Serialize(session.options); err == nil {
			args = append(args, sessionOptionField, options)
		} else {
			return err
		}
	}

	args = append(args, sessionLastAccessedField, session.lastAccessed.Unix())
	hsetCmd := s.connection.HSet(context.Background(), key, args...)
	if hsetCmd.Err() != nil {
		return hsetCmd.Err()
	}

	exp := session.expiration()
	expCmd := s.connection.ExpireAt(context.Background(), key, exp)
	return expCmd.Err()
}

func getKey(name string, id string) string {
	return fmt.Sprintf("%s:%s", name, id)
}

func Serialize(src interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(src); err != nil {
		return nil, errors.Wrap(err, "Cannot serialize value")
	}
	return buf.Bytes(), nil
}

func Deserialize(src io.Reader, dst interface{}) error {
	dec := gob.NewDecoder(src)
	if err := dec.Decode(dst); err != nil {
		return errors.Wrap(err, "Cannot serialize value")
	}
	return nil
}

