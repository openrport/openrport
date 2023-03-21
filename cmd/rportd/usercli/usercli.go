package usercli

import "fmt"

func CreateUser(userService UserService, pr PasswordReader, username string, groups []string, twoFASendTo string) error {
	password, err := promptForPassword(pr)
	if err != nil {
		return err
	}
	err = userService.Create(UserInput{
		Username:    username,
		Password:    password,
		Groups:      groups,
		TwoFASendTo: twoFASendTo,
	})
	if err != nil {
		return fmt.Errorf("Could not add user: %w", err)
	}
	fmt.Println("User added successfully.")

	return nil
}

func UpdateUser(userService UserService, pr PasswordReader, username string, groups []string, twoFASendTo string, askForPassword bool) error {
	var password string
	var err error
	if askForPassword {
		password, err = promptForPassword(pr)
		if err != nil {
			return err
		}
	}
	err = userService.Change(UserInput{
		Username:    username,
		Password:    password,
		Groups:      groups,
		TwoFASendTo: twoFASendTo,
	})
	if err != nil {
		return fmt.Errorf("Could not change user: %w", err)
	}
	fmt.Println("User changed successfully.")

	return nil
}

func DeleteUser(userService UserService, username string) error {
	err := userService.Delete(username)
	if err != nil {
		return fmt.Errorf("Could not delete user: %w", err)
	}
	fmt.Println("User deleted successfully.")

	return nil
}
