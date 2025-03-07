package orgimpl

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/setting"
)

func TestIntegrationOrgDataAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ss := db.InitTestDB(t)
	orgStore := sqlStore{
		db:      ss,
		dialect: ss.GetDialect(),
	}

	t.Run("org not found", func(t *testing.T) {
		_, err := orgStore.Get(context.Background(), 1)
		require.Error(t, err, org.ErrOrgNotFound)
	})

	t.Run("org inserted", func(t *testing.T) {
		_, err := orgStore.Insert(context.Background(), &org.Org{
			Version: 1,
			Name:    "test1",
			Created: time.Now(),
			Updated: time.Now(),
		})
		require.NoError(t, err)
	})

	t.Run("org inserted with next available org ID", func(t *testing.T) {
		orgID, err := orgStore.Insert(context.Background(), &org.Org{
			ID:      55,
			Version: 1,
			Name:    "test2",
			Created: time.Now(),
			Updated: time.Now(),
		})
		require.NoError(t, err)
		_, err = orgStore.Get(context.Background(), orgID)
		require.NoError(t, err)
	})

	t.Run("delete by user", func(t *testing.T) {
		err := orgStore.DeleteUserFromAll(context.Background(), 1)
		require.NoError(t, err)
	})

	t.Run("Update org address", func(t *testing.T) {
		// make sure ac2 has no org
		ac2 := &org.Org{ID: 21, Name: "name", Version: 1, Created: time.Now(), Updated: time.Now()}
		_, err := orgStore.Insert(context.Background(), ac2)
		require.NoError(t, err)
		err = orgStore.UpdateAddress(context.Background(), &org.UpdateOrgAddressCommand{
			OrgID: ac2.ID,
			Address: org.Address{
				Address1: "address1",
				Address2: "address2",
				City:     "city",
				ZipCode:  "zip",
				State:    "state",
				Country:  "country"},
		})
		require.NoError(t, err)
		orga, err := orgStore.Get(context.Background(), ac2.ID)
		require.NoError(t, err)
		require.Equal(t, "address1", orga.Address1)
	})

	t.Run("Removing org", func(t *testing.T) {
		// make sure ac2 has no org
		ac2 := &org.Org{ID: 22, Name: "ac2", Version: 1, Created: time.Now(), Updated: time.Now()}
		_, err := orgStore.Insert(context.Background(), ac2)
		require.NoError(t, err)
		err = orgStore.Delete(context.Background(), &org.DeleteOrgCommand{ID: ac2.ID})
		require.NoError(t, err)

		// TODO: this part of the test will be added when we move RemoveOrgUser to org store
		// "Removing user from org should delete user completely if in no other org"
		// // remove ac2 user from ac1 org
		// remCmd := models.RemoveOrgUserCommand{OrgId: ac1.OrgID, UserId: ac2.ID, ShouldDeleteOrphanedUser: true}
		// err = orgStore.RemoveOrgUser(context.Background(), &remCmd)
		// require.NoError(t, err)
		// require.True(t, remCmd.UserWasDeleted)

		// err = orgStore.GetSignedInUser(context.Background(), &models.GetSignedInUserQuery{UserId: ac2.ID})
		// require.Equal(t, err, user.ErrUserNotFound)
	})

	t.Run("Given we have organizations, we can query them by IDs", func(t *testing.T) {
		var err error
		var cmd *org.CreateOrgCommand
		ids := []int64{}

		for i := 1; i < 4; i++ {
			cmd = &org.CreateOrgCommand{Name: fmt.Sprint("Org #", i)}
			result, err := orgStore.CreateWithMember(context.Background(), cmd)
			require.NoError(t, err)

			ids = append(ids, result.ID)
		}

		query := &org.SearchOrgsQuery{IDs: ids}
		result, err := orgStore.Search(context.Background(), query)
		require.NoError(t, err)
		assert.Equal(t, 3, len(result))
	})

	t.Run("Given we have organizations, we can limit and paginate search", func(t *testing.T) {
		ss = db.InitTestDB(t)
		for i := 1; i < 4; i++ {
			cmd := &org.CreateOrgCommand{Name: fmt.Sprint("Orga #", i)}
			_, err := orgStore.CreateWithMember(context.Background(), cmd)
			require.NoError(t, err)
		}

		t.Run("Should be able to search with defaults", func(t *testing.T) {
			query := &org.SearchOrgsQuery{}
			result, err := orgStore.Search(context.Background(), query)

			require.NoError(t, err)
			assert.Equal(t, 3, len(result))
		})

		t.Run("Should be able to limit search", func(t *testing.T) {
			query := &org.SearchOrgsQuery{Limit: 1}
			result, err := orgStore.Search(context.Background(), query)

			require.NoError(t, err)
			assert.Equal(t, 1, len(result))
		})

		t.Run("Should be able to limit and paginate search", func(t *testing.T) {
			query := &org.SearchOrgsQuery{Limit: 2, Page: 1}
			result, err := orgStore.Search(context.Background(), query)

			require.NoError(t, err)
			assert.Equal(t, 1, len(result))
		})

		t.Run("Get org by ID", func(t *testing.T) {
			query := &org.GetOrgByIdQuery{ID: 1}
			result, err := orgStore.GetByID(context.Background(), query)

			require.NoError(t, err)
			assert.Equal(t, "Orga #1", result.Name)
		})

		t.Run("Get org by handler name", func(t *testing.T) {
			query := &org.GetOrgByNameQuery{Name: "Orga #1"}
			result, err := orgStore.GetByName(context.Background(), query)

			require.NoError(t, err)
			assert.Equal(t, int64(1), result.ID)
		})
	})

	t.Run("Testing Account DB Access", func(t *testing.T) {
		sqlStore := db.InitTestDB(t)

		t.Run("Given we have organizations, we can query them by IDs", func(t *testing.T) {
			var err error
			var cmd *models.CreateOrgCommand
			ids := []int64{}

			for i := 1; i < 4; i++ {
				cmd = &models.CreateOrgCommand{Name: fmt.Sprint("Org #", i)}
				err = sqlStore.CreateOrg(context.Background(), cmd)
				require.NoError(t, err)

				ids = append(ids, cmd.Result.Id)
			}

			query := &org.SearchOrgsQuery{IDs: ids}
			queryResult, err := orgStore.Search(context.Background(), query)

			require.NoError(t, err)
			require.Equal(t, len(queryResult), 3)
		})

		t.Run("Given we have organizations, we can limit and paginate search", func(t *testing.T) {
			sqlStore = db.InitTestDB(t)
			for i := 1; i < 4; i++ {
				cmd := &models.CreateOrgCommand{Name: fmt.Sprint("Org #", i)}
				err := sqlStore.CreateOrg(context.Background(), cmd)
				require.NoError(t, err)
			}

			t.Run("Should be able to search with defaults", func(t *testing.T) {
				query := &org.SearchOrgsQuery{}
				queryResult, err := orgStore.Search(context.Background(), query)

				require.NoError(t, err)
				require.Equal(t, len(queryResult), 3)
			})

			t.Run("Should be able to limit search", func(t *testing.T) {
				query := &org.SearchOrgsQuery{Limit: 1}
				queryResult, err := orgStore.Search(context.Background(), query)

				require.NoError(t, err)
				require.Equal(t, len(queryResult), 1)
			})

			t.Run("Should be able to limit and paginate search", func(t *testing.T) {
				query := &org.SearchOrgsQuery{Limit: 2, Page: 1}
				queryResult, err := orgStore.Search(context.Background(), query)

				require.NoError(t, err)
				require.Equal(t, len(queryResult), 1)
			})
		})
	})
}

func TestIntegrationOrgUserDataAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ss := db.InitTestDB(t)
	orgUserStore := sqlStore{
		db:      ss,
		dialect: ss.GetDialect(),
		cfg:     setting.NewCfg(),
	}

	t.Run("org user inserted", func(t *testing.T) {
		_, err := orgUserStore.InsertOrgUser(context.Background(), &org.OrgUser{
			ID:      1,
			OrgID:   1,
			UserID:  1,
			Created: time.Now(),
			Updated: time.Now(),
		})
		require.NoError(t, err)
	})

	t.Run("delete by user", func(t *testing.T) {
		err := orgUserStore.DeleteUserFromAll(context.Background(), 1)
		require.NoError(t, err)
	})

	// TODO: these test will be added when store will be CRUD
	// t.Run("Given single org mode", func(t *testing.T) {
	// 	sqlStore.Cfg.AutoAssignOrg = true
	// 	sqlStore.Cfg.AutoAssignOrgId = 1
	// 	sqlStore.Cfg.AutoAssignOrgRole = "Viewer"

	// 	t.Run("Users should be added to default organization", func(t *testing.T) {
	// 		ac1cmd := user.CreateUserCommand{Login: "ac1", Email: "ac1@test.com", Name: "ac1 name"}
	// 		ac2cmd := user.CreateUserCommand{Login: "ac2", Email: "ac2@test.com", Name: "ac2 name"}

	// 		ac1, err := sqlStore.CreateUser(context.Background(), ac1cmd)
	// 		require.NoError(t, err)
	// 		ac2, err := sqlStore.CreateUser(context.Background(), ac2cmd)
	// 		require.NoError(t, err)

	// 		q1 := models.GetUserOrgListQuery{UserId: ac1.ID}
	// 		q2 := models.GetUserOrgListQuery{UserId: ac2.ID}
	// 		err = sqlStore.GetUserOrgList(context.Background(), &q1)
	// 		require.NoError(t, err)
	// 		err = sqlStore.GetUserOrgList(context.Background(), &q2)
	// 		require.NoError(t, err)

	// 		require.Equal(t, q1.Result[0].OrgId, q2.Result[0].OrgId)
	// 		require.Equal(t, string(q1.Result[0].Role), "Viewer")
	// 	})
	// })

	// t.Run("Can get user organizations", func(t *testing.T) {
	// 	query := models.GetUserOrgListQuery{UserId: ac2.ID}
	// 	err := sqlStore.GetUserOrgList(context.Background(), &query)

	// 	require.NoError(t, err)
	// 	require.Equal(t, len(query.Result), 2)
	// })

	t.Run("Update org users", func(t *testing.T) {
		_, err := orgUserStore.InsertOrgUser(context.Background(), &org.OrgUser{
			ID:      1,
			OrgID:   1,
			UserID:  1,
			Created: time.Now(),
			Updated: time.Now(),
		})
		require.NoError(t, err)

		cmd := &org.UpdateOrgUserCommand{
			Role:   org.RoleAdmin,
			UserID: 1,
			OrgID:  1,
		}
		err = orgUserStore.UpdateOrgUser(context.Background(), cmd)
		require.NoError(t, err)
	})
	t.Run("GetOrgUsers and UpdateOrgUsers", func(t *testing.T) {
		ss := db.InitTestDB(t)
		ac1cmd := user.CreateUserCommand{Login: "ac1", Email: "ac1@test.com", Name: "ac1 name"}
		ac2cmd := user.CreateUserCommand{Login: "ac2", Email: "ac2@test.com", Name: "ac2 name", IsAdmin: true}
		ac1, err := ss.CreateUser(context.Background(), ac1cmd)
		require.NoError(t, err)
		ac2, err := ss.CreateUser(context.Background(), ac2cmd)
		require.NoError(t, err)
		cmd := org.AddOrgUserCommand{
			OrgID:  ac1.OrgID,
			UserID: ac2.ID,
			Role:   org.RoleViewer,
		}

		err = orgUserStore.AddOrgUser(context.Background(), &cmd)
		require.NoError(t, err)

		t.Run("Can update org user role", func(t *testing.T) {
			updateCmd := org.UpdateOrgUserCommand{OrgID: ac1.OrgID, UserID: ac2.ID, Role: org.RoleAdmin}
			err = orgUserStore.UpdateOrgUser(context.Background(), &updateCmd)
			require.NoError(t, err)

			orgUsersQuery := org.GetOrgUsersQuery{
				OrgID: ac1.OrgID,
				User: &user.SignedInUser{
					OrgID:       ac1.OrgID,
					Permissions: map[int64]map[string][]string{ac1.OrgID: {accesscontrol.ActionOrgUsersRead: {accesscontrol.ScopeUsersAll}}},
				},
			}
			result, err := orgUserStore.GetOrgUsers(context.Background(), &orgUsersQuery)
			require.NoError(t, err)

			require.EqualValues(t, result[1].Role, org.RoleAdmin)
		})
		t.Run("Can get organization users", func(t *testing.T) {
			query := org.GetOrgUsersQuery{
				OrgID: ac1.OrgID,
				User: &user.SignedInUser{
					OrgID:       ac1.OrgID,
					Permissions: map[int64]map[string][]string{ac1.OrgID: {accesscontrol.ActionOrgUsersRead: {accesscontrol.ScopeUsersAll}}},
				},
			}
			result, err := orgUserStore.GetOrgUsers(context.Background(), &query)

			require.NoError(t, err)
			require.Equal(t, len(result), 2)
			require.Equal(t, result[0].Role, "Admin")
		})

		t.Run("Can get organization users with query", func(t *testing.T) {
			query := org.GetOrgUsersQuery{
				OrgID: ac1.OrgID,
				Query: "ac1",
				User: &user.SignedInUser{
					OrgID:       ac1.OrgID,
					Permissions: map[int64]map[string][]string{ac1.OrgID: {accesscontrol.ActionOrgUsersRead: {accesscontrol.ScopeUsersAll}}},
				},
			}
			result, err := orgUserStore.GetOrgUsers(context.Background(), &query)

			require.NoError(t, err)
			require.Equal(t, len(result), 1)
			require.Equal(t, result[0].Email, ac1.Email)
		})
		t.Run("Can get organization users with query and limit", func(t *testing.T) {
			query := org.GetOrgUsersQuery{
				OrgID: ac1.OrgID,
				Query: "ac",
				Limit: 1,
				User: &user.SignedInUser{
					OrgID:       ac1.OrgID,
					Permissions: map[int64]map[string][]string{ac1.OrgID: {accesscontrol.ActionOrgUsersRead: {accesscontrol.ScopeUsersAll}}},
				},
			}
			result, err := orgUserStore.GetOrgUsers(context.Background(), &query)

			require.NoError(t, err)
			require.Equal(t, len(result), 1)
			require.Equal(t, result[0].Email, ac1.Email)
		})
		t.Run("Cannot update role so no one is admin user", func(t *testing.T) {
			remCmd := org.RemoveOrgUserCommand{OrgID: ac1.OrgID, UserID: ac2.ID, ShouldDeleteOrphanedUser: true}
			err := orgUserStore.RemoveOrgUser(context.Background(), &remCmd)
			require.NoError(t, err)
			cmd := org.UpdateOrgUserCommand{OrgID: ac1.OrgID, UserID: ac1.ID, Role: org.RoleViewer}
			err = orgUserStore.UpdateOrgUser(context.Background(), &cmd)
			require.Equal(t, models.ErrLastOrgAdmin, err)
		})

		t.Run("Removing user from org should delete user completely if in no other org", func(t *testing.T) {
			// make sure ac2 has no org
			err := orgUserStore.Delete(context.Background(), &org.DeleteOrgCommand{ID: ac2.OrgID})
			require.NoError(t, err)

			// remove ac2 user from ac1 org
			remCmd := org.RemoveOrgUserCommand{OrgID: ac1.OrgID, UserID: ac2.ID, ShouldDeleteOrphanedUser: true}
			err = orgUserStore.RemoveOrgUser(context.Background(), &remCmd)
			require.NoError(t, err)
			require.True(t, remCmd.UserWasDeleted)

			err = ss.GetSignedInUser(context.Background(), &models.GetSignedInUserQuery{UserId: ac2.ID})
			require.Equal(t, err, user.ErrUserNotFound)
		})

		t.Run("Cannot delete last admin org user", func(t *testing.T) {
			cmd := org.RemoveOrgUserCommand{OrgID: ac1.OrgID, UserID: ac1.ID}
			err := orgUserStore.RemoveOrgUser(context.Background(), &cmd)
			require.Equal(t, err, models.ErrLastOrgAdmin)
		})
	})

	t.Run("Given single org and 2 users inserted", func(t *testing.T) {
		ss = db.InitTestDB(t)
		testUser := &user.SignedInUser{
			Permissions: map[int64]map[string][]string{
				1: {accesscontrol.ActionOrgUsersRead: []string{accesscontrol.ScopeUsersAll}},
			},
		}
		ss.Cfg.AutoAssignOrg = true
		ss.Cfg.AutoAssignOrgId = 1
		ss.Cfg.AutoAssignOrgRole = "Viewer"

		ac1cmd := user.CreateUserCommand{Login: "ac1", Email: "ac1@test.com", Name: "ac1 name"}
		ac2cmd := user.CreateUserCommand{Login: "ac2", Email: "ac2@test.com", Name: "ac2 name"}

		ac1, err := ss.CreateUser(context.Background(), ac1cmd)
		testUser.OrgID = ac1.OrgID
		require.NoError(t, err)
		_, err = ss.CreateUser(context.Background(), ac2cmd)
		require.NoError(t, err)

		t.Run("Can get organization users paginated with query", func(t *testing.T) {
			query := org.SearchOrgUsersQuery{
				OrgID: ac1.OrgID,
				Page:  1,
				User:  testUser,
			}
			result, err := orgUserStore.SearchOrgUsers(context.Background(), &query)
			require.NoError(t, err)
			require.Equal(t, 2, len(result.OrgUsers))
		})

		t.Run("Can get organization users paginated and limited", func(t *testing.T) {
			query := org.SearchOrgUsersQuery{
				OrgID: ac1.OrgID,
				Limit: 1,
				Page:  1,
				User:  testUser,
			}
			result, err := orgUserStore.SearchOrgUsers(context.Background(), &query)
			require.NoError(t, err)
			require.Equal(t, 1, len(result.OrgUsers))
		})
	})
}

// This test will be refactore after the CRUD store  refactor
func TestIntegrationSQLStore_AddOrgUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	store := db.InitTestDB(t)
	store.Cfg.AutoAssignOrg = true
	store.Cfg.AutoAssignOrgId = 1
	store.Cfg.AutoAssignOrgRole = "Viewer"
	orgUserStore := sqlStore{
		db:      store,
		dialect: store.GetDialect(),
		cfg:     setting.NewCfg(),
	}

	// create org and admin
	u, err := store.CreateUser(context.Background(), user.CreateUserCommand{
		Login: "admin",
	})
	require.NoError(t, err)

	// create a service account with no org
	sa, err := store.CreateUser(context.Background(), user.CreateUserCommand{
		Login:            "sa-no-org",
		IsServiceAccount: true,
		SkipOrgSetup:     true,
	})

	require.NoError(t, err)
	require.Equal(t, int64(-1), sa.OrgID)

	// assign the sa to the org but without the override. should fail
	err = orgUserStore.AddOrgUser(context.Background(), &org.AddOrgUserCommand{
		Role:   "Viewer",
		OrgID:  u.OrgID,
		UserID: sa.ID,
	})
	require.Error(t, err)

	// assign the sa to the org with the override. should succeed
	err = orgUserStore.AddOrgUser(context.Background(), &org.AddOrgUserCommand{
		Role:                      "Viewer",
		OrgID:                     u.OrgID,
		UserID:                    sa.ID,
		AllowAddingServiceAccount: true,
	})

	require.NoError(t, err)

	// assert the org has been correctly set
	saFound := new(user.User)
	err = store.WithDbSession(context.Background(), func(sess *db.Session) error {
		has, err := sess.ID(sa.ID).Get(saFound)
		if err != nil {
			return err
		} else if !has {
			return user.ErrUserNotFound
		}
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, saFound.OrgID, u.OrgID)
}

func TestIntegration_SQLStore_GetOrgUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	tests := []struct {
		desc             string
		query            *org.GetOrgUsersQuery
		expectedNumUsers int
	}{
		{
			desc: "should return all users",
			query: &org.GetOrgUsersQuery{
				OrgID: 1,
				User: &user.SignedInUser{
					OrgID:       1,
					Permissions: map[int64]map[string][]string{1: {accesscontrol.ActionOrgUsersRead: {accesscontrol.ScopeUsersAll}}},
				},
			},
			expectedNumUsers: 10,
		},
		{
			desc: "should return no users",
			query: &org.GetOrgUsersQuery{
				OrgID: 1,
				User: &user.SignedInUser{
					OrgID:       1,
					Permissions: map[int64]map[string][]string{1: {accesscontrol.ActionOrgUsersRead: {""}}},
				},
			},
			expectedNumUsers: 0,
		},
		{
			desc: "should return some users",
			query: &org.GetOrgUsersQuery{
				OrgID: 1,
				User: &user.SignedInUser{
					OrgID: 1,
					Permissions: map[int64]map[string][]string{1: {accesscontrol.ActionOrgUsersRead: {
						"users:id:1",
						"users:id:5",
						"users:id:9",
					}}},
				},
			},
			expectedNumUsers: 3,
		},
	}

	store := db.InitTestDB(t)
	orgUserStore := sqlStore{
		db:      store,
		dialect: store.GetDialect(),
		cfg:     setting.NewCfg(),
	}
	orgUserStore.cfg.IsEnterprise = true
	defer func() {
		orgUserStore.cfg.IsEnterprise = false
	}()
	store.Cfg = setting.NewCfg()
	seedOrgUsers(t, &orgUserStore, store, 10)

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result, err := orgUserStore.GetOrgUsers(context.Background(), tt.query)
			require.NoError(t, err)
			require.Len(t, result, tt.expectedNumUsers)

			if !hasWildcardScope(tt.query.User, accesscontrol.ActionOrgUsersRead) {
				for _, u := range result {
					assert.Contains(t, tt.query.User.Permissions[tt.query.User.OrgID][accesscontrol.ActionOrgUsersRead], fmt.Sprintf("users:id:%d", u.UserID))
				}
			}
		})
	}
}

func seedOrgUsers(t *testing.T, orgUserStore store, store *sqlstore.SQLStore, numUsers int) {
	t.Helper()
	// Seed users
	for i := 1; i <= numUsers; i++ {
		user, err := store.CreateUser(context.Background(), user.CreateUserCommand{
			Login: fmt.Sprintf("user-%d", i),
			OrgID: 1,
		})
		require.NoError(t, err)

		if i != 1 {
			err = orgUserStore.AddOrgUser(context.Background(), &org.AddOrgUserCommand{
				Role:   "Viewer",
				OrgID:  1,
				UserID: user.ID,
			})
			require.NoError(t, err)
		}
	}
}

func hasWildcardScope(user *user.SignedInUser, action string) bool {
	for _, scope := range user.Permissions[user.OrgID][action] {
		if strings.HasSuffix(scope, ":*") {
			return true
		}
	}
	return false
}

func TestIntegration_SQLStore_GetOrgUsers_PopulatesCorrectly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	// The millisecond part is not stored in the DB
	constNow := time.Date(2022, 8, 17, 20, 34, 58, 0, time.UTC)
	sqlstore.MockTimeNow(constNow)
	defer sqlstore.ResetTimeNow()

	store := db.InitTestDB(t, sqlstore.InitTestDBOpt{})
	orgUserStore := sqlStore{
		db:      store,
		dialect: store.GetDialect(),
		cfg:     setting.NewCfg(),
	}

	id, err := orgUserStore.Insert(context.Background(),
		&org.Org{
			ID:      1,
			Created: constNow,
			Updated: constNow,
		})
	require.NoError(t, err)

	newUser, err := store.CreateUser(context.Background(), user.CreateUserCommand{
		Login:      "Viewer",
		Email:      "viewer@localhost",
		OrgID:      id,
		IsDisabled: true,
		Name:       "Viewer Localhost",
	})
	require.NoError(t, err)

	err = orgUserStore.AddOrgUser(context.Background(), &org.AddOrgUserCommand{
		Role:   "Viewer",
		OrgID:  1,
		UserID: newUser.ID,
	})
	require.NoError(t, err)

	query := &org.GetOrgUsersQuery{
		OrgID:  1,
		UserID: newUser.ID,
		User: &user.SignedInUser{
			OrgID:       1,
			Permissions: map[int64]map[string][]string{1: {accesscontrol.ActionOrgUsersRead: {accesscontrol.ScopeUsersAll}}},
		},
	}
	result, err := orgUserStore.GetOrgUsers(context.Background(), query)
	require.NoError(t, err)
	require.Len(t, result, 1)

	actual := result[0]
	assert.Equal(t, int64(1), actual.OrgID)
	assert.Equal(t, int64(1), actual.UserID)
	assert.Equal(t, "viewer@localhost", actual.Email)
	assert.Equal(t, "Viewer Localhost", actual.Name)
	assert.Equal(t, "Viewer", actual.Login)
	assert.Equal(t, "Viewer", actual.Role)
	assert.Equal(t, constNow.AddDate(-10, 0, 0), actual.LastSeenAt)
	assert.Equal(t, constNow, actual.Created)
	assert.Equal(t, constNow, actual.Updated)
	assert.Equal(t, true, actual.IsDisabled)
}

func TestIntegration_SQLStore_SearchOrgUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	tests := []struct {
		desc             string
		query            *org.SearchOrgUsersQuery
		expectedNumUsers int
	}{
		{
			desc: "should return all users",
			query: &org.SearchOrgUsersQuery{
				OrgID: 1,
				User: &user.SignedInUser{
					OrgID:       1,
					Permissions: map[int64]map[string][]string{1: {accesscontrol.ActionOrgUsersRead: {accesscontrol.ScopeUsersAll}}},
				},
			},
			expectedNumUsers: 10,
		},
		{
			desc: "should return no users",
			query: &org.SearchOrgUsersQuery{
				OrgID: 1,
				User: &user.SignedInUser{
					OrgID:       1,
					Permissions: map[int64]map[string][]string{1: {accesscontrol.ActionOrgUsersRead: {""}}},
				},
			},
			expectedNumUsers: 0,
		},
		{
			desc: "should return some users",
			query: &org.SearchOrgUsersQuery{
				OrgID: 1,
				User: &user.SignedInUser{
					OrgID: 1,
					Permissions: map[int64]map[string][]string{1: {accesscontrol.ActionOrgUsersRead: {
						"users:id:1",
						"users:id:5",
						"users:id:9",
					}}},
				},
			},
			expectedNumUsers: 3,
		},
	}

	store := db.InitTestDB(t, sqlstore.InitTestDBOpt{})
	orgUserStore := sqlStore{
		db:      store,
		dialect: store.GetDialect(),
		cfg:     setting.NewCfg(),
	}
	// orgUserStore.cfg.Skip
	seedOrgUsers(t, &orgUserStore, store, 10)

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result, err := orgUserStore.SearchOrgUsers(context.Background(), tt.query)
			fmt.Println("users:", result)

			require.NoError(t, err)
			assert.Len(t, result.OrgUsers, tt.expectedNumUsers)

			if !hasWildcardScope(tt.query.User, accesscontrol.ActionOrgUsersRead) {
				for _, u := range result.OrgUsers {
					assert.Contains(t, tt.query.User.Permissions[tt.query.User.OrgID][accesscontrol.ActionOrgUsersRead], fmt.Sprintf("users:id:%d", u.UserID))
				}
			}
		})
	}
}

func TestIntegration_SQLStore_RemoveOrgUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	store := db.InitTestDB(t)
	orgUserStore := sqlStore{
		db:      store,
		dialect: store.GetDialect(),
		cfg:     setting.NewCfg(),
	}
	// create org and admin
	_, err := store.CreateUser(context.Background(), user.CreateUserCommand{
		Login: "admin",
		OrgID: 1,
	})
	require.NoError(t, err)

	// create a user with no org
	_, err = store.CreateUser(context.Background(), user.CreateUserCommand{
		Login:        "user",
		OrgID:        1,
		SkipOrgSetup: true,
	})
	require.NoError(t, err)

	// assign the user to the org
	err = orgUserStore.AddOrgUser(context.Background(), &org.AddOrgUserCommand{
		Role:   "Viewer",
		OrgID:  1,
		UserID: 2,
	})
	require.NoError(t, err)

	// remove the user org
	err = orgUserStore.RemoveOrgUser(context.Background(), &org.RemoveOrgUserCommand{
		UserID:                   2,
		OrgID:                    1,
		ShouldDeleteOrphanedUser: false,
	})
	require.NoError(t, err)
}
