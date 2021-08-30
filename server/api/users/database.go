package users

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	chshare "github.com/cloudradar-monitoring/rport/share"

	"github.com/jmoiron/sqlx"
)

type UserDatabase struct {
	db              *sqlx.DB
	usersTableName  string
	groupsTableName string
	twoFAOn         bool
	hasTokenColumn  bool
	logger          *chshare.Logger
}

func NewUserDatabase(DB *sqlx.DB, usersTableName, groupsTableName string, twoFAOn bool, logger *chshare.Logger) (*UserDatabase, error) {
	d := &UserDatabase{
		db:              DB,
		usersTableName:  usersTableName,
		groupsTableName: groupsTableName,
		twoFAOn:         twoFAOn,
		logger:          logger,
	}
	if err := d.checkDatabaseTables(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *UserDatabase) getSelectClause() string {
	s := "username, password"
	if d.twoFAOn {
		s += ", two_fa_send_to"
	}
	if d.hasTokenColumn {
		s += ", token"
	}
	return s
}

// todo use context for all db operations
func (d *UserDatabase) checkDatabaseTables() error {
	_, err := d.db.Exec(fmt.Sprintf("SELECT token FROM `%s` LIMIT 0", d.usersTableName))
	if err == nil {
		d.hasTokenColumn = true
	}
	_, err = d.db.Exec(fmt.Sprintf("SELECT %s FROM `%s` LIMIT 0", d.getSelectClause(), d.usersTableName))
	if err != nil {
		return err
	}
	_, err = d.db.Exec(fmt.Sprintf("SELECT username, `group` FROM `%s` LIMIT 0", d.groupsTableName))
	if err != nil {
		return err
	}
	return nil
}

// todo use context for all db operations
func (d *UserDatabase) GetByUsername(username string) (*User, error) {
	user := &User{}
	err := d.db.Get(user, fmt.Sprintf("SELECT %s FROM `%s` WHERE username = ? LIMIT 1", d.getSelectClause(), d.usersTableName), username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	err = d.db.Select(&user.Groups, fmt.Sprintf("SELECT DISTINCT(`group`) FROM `%s` WHERE username = ?", d.groupsTableName), username)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return user, nil
}

// todo use context for all db operations
func (d *UserDatabase) GetAll() ([]*User, error) {
	var usrs []*User
	err := d.db.Select(&usrs, fmt.Sprintf("SELECT %s FROM `%s` ORDER BY username", d.getSelectClause(), d.usersTableName))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var groups []struct {
		Username string `db:"username"`
		Group    string `db:"group"`
	}
	err = d.db.Select(&groups, fmt.Sprintf("SELECT `username`, `group` FROM `%s` ORDER BY `group`", d.groupsTableName))
	if err != nil {
		if err == sql.ErrNoRows {
			return usrs, nil
		}
		return nil, err
	}
	for i := range groups {
		for y := range usrs {
			if usrs[y].Username == groups[i].Username {
				usrs[y].Groups = append(usrs[y].Groups, groups[i].Group)
			}
		}
	}

	return usrs, nil
}

func (d *UserDatabase) GetAllGroups() ([]string, error) {
	var groups []string
	err := d.db.Select(&groups, fmt.Sprintf("SELECT DISTINCT `group` FROM `%s` ORDER BY `group`", d.groupsTableName))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return groups, nil
}

func (d *UserDatabase) handleRollback(tx *sqlx.Tx) {
	err := tx.Rollback()
	if err != nil {
		d.logger.Errorf("Failed to rollback transaction: %v", err)
	}
}

// todo use context for all db operations
func (d *UserDatabase) Add(usr *User) error {
	tx, err := d.db.Beginx()
	if err != nil {
		return err
	}

	if d.twoFAOn {
		_, err = tx.Exec(
			fmt.Sprintf("INSERT INTO `%s` (`username`, `password`, `two_fa_send_to`) VALUES (?, ?, ?)", d.usersTableName),
			usr.Username,
			usr.Password,
			usr.TwoFASendTo,
		)
	} else {
		_, err = tx.Exec(
			fmt.Sprintf("INSERT INTO `%s` (`username`, `password`) VALUES (?, ?)", d.usersTableName),
			usr.Username,
			usr.Password,
		)
	}
	if err != nil {
		d.handleRollback(tx)
		return err
	}

	for i := range usr.Groups {
		_, err := tx.Exec(
			fmt.Sprintf("INSERT INTO `%s` (`username`, `group`) VALUES (?, ?)", d.groupsTableName),
			usr.Username,
			usr.Groups[i],
		)
		if err != nil {
			d.handleRollback(tx)
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

// todo use context for all db operations
func (d *UserDatabase) Update(usr *User, usernameToUpdate string) error {
	if usernameToUpdate == "" {
		return errors.New("cannot update user with empty username")
	}

	params := []interface{}{}
	statements := []string{}
	if usr.Password != "" {
		statements = append(statements, "`password` = ?")
		params = append(params, usr.Password)
	}

	if usr.TwoFASendTo != "" {
		statements = append(statements, "`two_fa_send_to` = ?")
		params = append(params, usr.TwoFASendTo)
	}

	if usr.Username != "" && usr.Username != usernameToUpdate {
		statements = append(statements, "`username` = ?")
		params = append(params, usr.Username)
	}

	if usr.Token != nil {
		statements = append(statements, "`token` = ?")
		params = append(params, usr.Token)
	}

	tx, err := d.db.Beginx()
	if err != nil {
		return err
	}

	if len(params) > 0 {
		q := fmt.Sprintf(
			"UPDATE `%s` SET %s WHERE username = ?",
			d.usersTableName,
			strings.Join(statements, ", "),
		)
		params = append(params, usernameToUpdate)
		_, err := tx.Exec(q, params...)
		if err != nil {
			d.handleRollback(tx)
			return err
		}
	}

	if usr.Username != "" && usernameToUpdate != usr.Username {
		_, err := tx.Exec(
			fmt.Sprintf("UPDATE `%s` SET `username` = ? WHERE `username` = ?", d.groupsTableName),
			usr.Username,
			usernameToUpdate,
		)
		if err != nil {
			d.handleRollback(tx)
			return err
		}
	}

	groupUserName := usernameToUpdate
	if usr.Username != "" {
		groupUserName = usr.Username
	}

	if usr.Groups != nil {
		_, err := tx.Exec(fmt.Sprintf("DELETE FROM `%s` WHERE `username` = ?", d.groupsTableName), groupUserName)
		if err != nil {
			d.handleRollback(tx)
			return err
		}
	}

	for i := range usr.Groups {
		group := usr.Groups[i]
		_, err := tx.Exec(
			fmt.Sprintf("INSERT INTO `%s` (`username`, `group`) VALUES (?, ?)", d.groupsTableName),
			groupUserName,
			group,
		)
		if err != nil {
			d.handleRollback(tx)
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

// todo use context for all db operations
func (d *UserDatabase) Delete(usernameToDelete string) error {
	tx, err := d.db.Beginx()
	if err != nil {
		return err
	}

	_, err = tx.Exec(fmt.Sprintf("DELETE FROM `%s` WHERE `username` = ?", d.usersTableName), usernameToDelete)
	if err != nil {
		d.handleRollback(tx)
		return err
	}

	_, err = tx.Exec(fmt.Sprintf("DELETE FROM `%s` WHERE `username` = ?", d.groupsTableName), usernameToDelete)
	if err != nil {
		d.handleRollback(tx)
		return err
	}

	return tx.Commit()
}
