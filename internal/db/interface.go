package db

type Database interface {
    Close() error
    CheckUser(username, password string) bool
    AddUser(username, password string) error
}
var _ Database = (*DB)(nil)