package migratecmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestAutomigrateCollectionCreate(t *testing.T) {
	scenarios := []struct {
		lang             string
		expectedTemplate string
	}{
		{
			migratecmd.TemplateLangJS,
			`
migrate((db) => {
  const collection = unmarshal({
    "id": "new_id",
    "created": "2022-01-01 00:00:00.000Z",
    "updated": "2022-01-01 00:00:00.000Z",
    "name": "new_name",
    "type": "auth",
    "system": true,
    "schema": [],
    "listRule": "@request.auth.id != '' && created > 0 || 'backtick` + "`" + `test' = 0",
    "viewRule": "id = \"1\"",
    "createRule": null,
    "updateRule": null,
    "deleteRule": null,
    "options": {
      "allowEmailAuth": false,
      "allowOAuth2Auth": false,
      "allowUsernameAuth": false,
      "exceptEmailDomains": null,
      "manageRule": "created > 0",
      "minPasswordLength": 20,
      "onlyEmailDomains": null,
      "requireEmail": false
    }
  }, new Collection());

  return Dao(db).saveCollection(collection);
}, (db) => {
  const dao = new Dao(db);
  const collection = dao.findCollectionByNameOrId("new_id");

  return dao.deleteCollection(collection);
})
`,
		},
		{
			migratecmd.TemplateLangGo,
			`
package _test_migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		jsonData := ` + "`" + `{
			"id": "new_id",
			"created": "2022-01-01 00:00:00.000Z",
			"updated": "2022-01-01 00:00:00.000Z",
			"name": "new_name",
			"type": "auth",
			"system": true,
			"schema": [],
			"listRule": "@request.auth.id != '' && created > 0 || ` + "'backtick` + \"`\" + `test' = 0" + `",
			"viewRule": "id = \"1\"",
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {
				"allowEmailAuth": false,
				"allowOAuth2Auth": false,
				"allowUsernameAuth": false,
				"exceptEmailDomains": null,
				"manageRule": "created > 0",
				"minPasswordLength": 20,
				"onlyEmailDomains": null,
				"requireEmail": false
			}
		}` + "`" + `

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("new_id")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
`,
		},
	}

	for i, s := range scenarios {
		app, _ := tests.NewTestApp()
		defer app.Cleanup()

		migrationsDir := filepath.Join(app.DataDir(), "_test_migrations")

		migratecmd.MustRegister(app, nil, &migratecmd.Options{
			TemplateLang: s.lang,
			Automigrate:  true,
			Dir:          migrationsDir,
		})

		// @todo remove after collections cache is replaced
		app.Bootstrap()

		collection := &models.Collection{}
		collection.Id = "new_id"
		collection.Name = "new_name"
		collection.Type = models.CollectionTypeAuth
		collection.System = true
		collection.Created, _ = types.ParseDateTime("2022-01-01 00:00:00.000Z")
		collection.Updated = collection.Created
		collection.ListRule = types.Pointer("@request.auth.id != '' && created > 0 || 'backtick`test' = 0")
		collection.ViewRule = types.Pointer(`id = "1"`)
		collection.SetOptions(models.CollectionAuthOptions{
			ManageRule:        types.Pointer("created > 0"),
			MinPasswordLength: 20,
		})
		collection.MarkAsNew()

		if err := app.Dao().SaveCollection(collection); err != nil {
			t.Fatalf("[%d] Failed to save collection, got %v", i, err)
		}

		files, err := os.ReadDir(migrationsDir)
		if err != nil {
			t.Fatalf("[%d] Expected migrationsDir to be created, got: %v", i, err)
		}

		if total := len(files); total != 1 {
			t.Fatalf("[%d] Expected 1 file to be generated, got %d", i, total)
		}

		expectedName := "_created_new_name." + s.lang
		if !strings.Contains(files[0].Name(), expectedName) {
			t.Fatalf("[%d] Expected filename to contains %q, got %q", i, expectedName, files[0].Name())
		}

		fullPath := filepath.Join(migrationsDir, files[0].Name())
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("[%d] Failed to read the generated migration file: %v", i, err)
		}

		if v := strings.TrimSpace(string(content)); v != strings.TrimSpace(s.expectedTemplate) {
			t.Fatalf("[%d] Expected template \n%v \ngot \n%v", i, s.expectedTemplate, v)
		}
	}
}

func TestAutomigrateCollectionDelete(t *testing.T) {
	scenarios := []struct {
		lang             string
		expectedTemplate string
	}{
		{
			migratecmd.TemplateLangJS,
			`
migrate((db) => {
  const dao = new Dao(db);
  const collection = dao.findCollectionByNameOrId("test123");

  return dao.deleteCollection(collection);
}, (db) => {
  const collection = unmarshal({
    "id": "test123",
    "created": "2022-01-01 00:00:00.000Z",
    "updated": "2022-01-01 00:00:00.000Z",
    "name": "test456",
    "type": "auth",
    "system": false,
    "schema": [],
    "listRule": "@request.auth.id != '' && created > 0 || 'backtick` + "`" + `test' = 0",
    "viewRule": "id = \"1\"",
    "createRule": null,
    "updateRule": null,
    "deleteRule": null,
    "options": {
      "allowEmailAuth": false,
      "allowOAuth2Auth": false,
      "allowUsernameAuth": false,
      "exceptEmailDomains": null,
      "manageRule": "created > 0",
      "minPasswordLength": 20,
      "onlyEmailDomains": null,
      "requireEmail": false
    }
  }, new Collection());

  return Dao(db).saveCollection(collection);
})
`,
		},
		{
			migratecmd.TemplateLangGo,
			`
package _test_migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("test123")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	}, func(db dbx.Builder) error {
		jsonData := ` + "`" + `{
			"id": "test123",
			"created": "2022-01-01 00:00:00.000Z",
			"updated": "2022-01-01 00:00:00.000Z",
			"name": "test456",
			"type": "auth",
			"system": false,
			"schema": [],
			"listRule": "@request.auth.id != '' && created > 0 || ` + "'backtick` + \"`\" + `test' = 0" + `",
			"viewRule": "id = \"1\"",
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {
				"allowEmailAuth": false,
				"allowOAuth2Auth": false,
				"allowUsernameAuth": false,
				"exceptEmailDomains": null,
				"manageRule": "created > 0",
				"minPasswordLength": 20,
				"onlyEmailDomains": null,
				"requireEmail": false
			}
		}` + "`" + `

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	})
}
`,
		},
	}

	for i, s := range scenarios {
		app, _ := tests.NewTestApp()
		defer app.Cleanup()

		migrationsDir := filepath.Join(app.DataDir(), "_test_migrations")

		migratecmd.MustRegister(app, nil, &migratecmd.Options{
			TemplateLang: s.lang,
			Automigrate:  true,
			Dir:          migrationsDir,
		})

		// create dummy collection
		collection := &models.Collection{}
		collection.Id = "test123"
		collection.Name = "test456"
		collection.Type = models.CollectionTypeAuth
		collection.Created, _ = types.ParseDateTime("2022-01-01 00:00:00.000Z")
		collection.Updated = collection.Created
		collection.ListRule = types.Pointer("@request.auth.id != '' && created > 0 || 'backtick`test' = 0")
		collection.ViewRule = types.Pointer(`id = "1"`)
		collection.SetOptions(models.CollectionAuthOptions{
			ManageRule:        types.Pointer("created > 0"),
			MinPasswordLength: 20,
		})
		collection.MarkAsNew()

		// use different dao to avoid triggering automigrate while saving the dummy collection
		if err := daos.New(app.DB()).SaveCollection(collection); err != nil {
			t.Fatalf("[%d] Failed to save dummy collection, got %v", i, err)
		}

		// @todo remove after collections cache is replaced
		app.Bootstrap()

		// delete the newly created dummy collection
		if err := app.Dao().DeleteCollection(collection); err != nil {
			t.Fatalf("[%d] Failed to delete dummy collection, got %v", i, err)
		}

		files, err := os.ReadDir(migrationsDir)
		if err != nil {
			t.Fatalf("[%d] Expected migrationsDir to be created, got: %v", i, err)
		}

		if total := len(files); total != 1 {
			t.Fatalf("[%d] Expected 1 file to be generated, got %d", i, total)
		}

		expectedName := "_deleted_test456." + s.lang
		if !strings.Contains(files[0].Name(), expectedName) {
			t.Fatalf("[%d] Expected filename to contains %q, got %q", i, expectedName, files[0].Name())
		}

		fullPath := filepath.Join(migrationsDir, files[0].Name())
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("[%d] Failed to read the generated migration file: %v", i, err)
		}

		if v := strings.TrimSpace(string(content)); v != strings.TrimSpace(s.expectedTemplate) {
			t.Fatalf("[%d] Expected template \n%v \ngot \n%v", i, s.expectedTemplate, v)
		}
	}
}

func TestAutomigrateCollectionUpdate(t *testing.T) {
	scenarios := []struct {
		lang             string
		expectedTemplate string
	}{
		{
			migratecmd.TemplateLangJS,
			`
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("test123")

  collection.name = "test456_update"
  collection.type = "base"
  collection.listRule = null
  collection.deleteRule = "updated > 0 && @request.auth.id != ''"
  collection.options = {}

  // remove
  collection.schema.removeField("f3_id")

  // add
  collection.schema.addField(unmarshal({
    "system": false,
    "id": "f4_id",
    "name": "f4_name",
    "type": "text",
    "required": false,
    "unique": false,
    "options": {
      "min": null,
      "max": null,
      "pattern": "` + "`" + `test backtick` + "`" + `123"
    }
  }, new SchemaField()))

  // update
  collection.schema.addField(unmarshal({
    "system": false,
    "id": "f2_id",
    "name": "f2_name_new",
    "type": "number",
    "required": false,
    "unique": true,
    "options": {
      "min": 10,
      "max": null
    }
  }, new SchemaField()))

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("test123")

  collection.name = "test456"
  collection.type = "auth"
  collection.listRule = "@request.auth.id != '' && created > 0"
  collection.deleteRule = null
  collection.options = {
    "allowEmailAuth": false,
    "allowOAuth2Auth": false,
    "allowUsernameAuth": false,
    "exceptEmailDomains": null,
    "manageRule": "created > 0",
    "minPasswordLength": 20,
    "onlyEmailDomains": null,
    "requireEmail": false
  }

  // add
  collection.schema.addField(unmarshal({
    "system": false,
    "id": "f3_id",
    "name": "f3_name",
    "type": "bool",
    "required": false,
    "unique": false,
    "options": {}
  }, new SchemaField()))

  // remove
  collection.schema.removeField("f4_id")

  // update
  collection.schema.addField(unmarshal({
    "system": false,
    "id": "f2_id",
    "name": "f2_name",
    "type": "number",
    "required": false,
    "unique": true,
    "options": {
      "min": 10,
      "max": null
    }
  }, new SchemaField()))

  return dao.saveCollection(collection)
})
`,
		},
		{
			migratecmd.TemplateLangGo,
			`
package _test_migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("test123")
		if err != nil {
			return err
		}

		collection.Name = "test456_update"

		collection.Type = "base"

		collection.ListRule = nil

		collection.DeleteRule = types.Pointer("updated > 0 && @request.auth.id != ''")

		options := map[string]any{}
		json.Unmarshal([]byte(` + "`" + `{}` + "`" + `), &options)
		collection.SetOptions(options)

		// remove
		collection.Schema.RemoveField("f3_id")

		// add
		new_f4_name := &schema.SchemaField{}
		json.Unmarshal([]byte(` + "`" + `{
			"system": false,
			"id": "f4_id",
			"name": "f4_name",
			"type": "text",
			"required": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ` + "\"` + \"`\" + `test backtick` + \"`\" + `123\"" + `
			}
		}` + "`" + `), new_f4_name)
		collection.Schema.AddField(new_f4_name)

		// update
		edit_f2_name_new := &schema.SchemaField{}
		json.Unmarshal([]byte(` + "`" + `{
			"system": false,
			"id": "f2_id",
			"name": "f2_name_new",
			"type": "number",
			"required": false,
			"unique": true,
			"options": {
				"min": 10,
				"max": null
			}
		}` + "`" + `), edit_f2_name_new)
		collection.Schema.AddField(edit_f2_name_new)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("test123")
		if err != nil {
			return err
		}

		collection.Name = "test456"

		collection.Type = "auth"

		collection.ListRule = types.Pointer("@request.auth.id != '' && created > 0")

		collection.DeleteRule = nil

		options := map[string]any{}
		json.Unmarshal([]byte(` + "`" + `{
			"allowEmailAuth": false,
			"allowOAuth2Auth": false,
			"allowUsernameAuth": false,
			"exceptEmailDomains": null,
			"manageRule": "created > 0",
			"minPasswordLength": 20,
			"onlyEmailDomains": null,
			"requireEmail": false
		}` + "`" + `), &options)
		collection.SetOptions(options)

		// add
		del_f3_name := &schema.SchemaField{}
		json.Unmarshal([]byte(` + "`" + `{
			"system": false,
			"id": "f3_id",
			"name": "f3_name",
			"type": "bool",
			"required": false,
			"unique": false,
			"options": {}
		}` + "`" + `), del_f3_name)
		collection.Schema.AddField(del_f3_name)

		// remove
		collection.Schema.RemoveField("f4_id")

		// update
		edit_f2_name_new := &schema.SchemaField{}
		json.Unmarshal([]byte(` + "`" + `{
			"system": false,
			"id": "f2_id",
			"name": "f2_name",
			"type": "number",
			"required": false,
			"unique": true,
			"options": {
				"min": 10,
				"max": null
			}
		}` + "`" + `), edit_f2_name_new)
		collection.Schema.AddField(edit_f2_name_new)

		return dao.SaveCollection(collection)
	})
}
`,
		},
	}

	for i, s := range scenarios {
		app, _ := tests.NewTestApp()
		defer app.Cleanup()

		migrationsDir := filepath.Join(app.DataDir(), "_test_migrations")

		migratecmd.MustRegister(app, nil, &migratecmd.Options{
			TemplateLang: s.lang,
			Automigrate:  true,
			Dir:          migrationsDir,
		})

		// create dummy collection
		collection := &models.Collection{}
		collection.Id = "test123"
		collection.Name = "test456"
		collection.Type = models.CollectionTypeAuth
		collection.Created, _ = types.ParseDateTime("2022-01-01 00:00:00.000Z")
		collection.Updated = collection.Created
		collection.ListRule = types.Pointer("@request.auth.id != '' && created > 0")
		collection.ViewRule = types.Pointer(`id = "1"`)
		collection.SetOptions(models.CollectionAuthOptions{
			ManageRule:        types.Pointer("created > 0"),
			MinPasswordLength: 20,
		})
		collection.MarkAsNew()
		collection.Schema.AddField(&schema.SchemaField{
			Id:       "f1_id",
			Name:     "f1_name",
			Type:     schema.FieldTypeText,
			Required: true,
		})
		collection.Schema.AddField(&schema.SchemaField{
			Id:     "f2_id",
			Name:   "f2_name",
			Type:   schema.FieldTypeNumber,
			Unique: true,
			Options: &schema.NumberOptions{
				Min: types.Pointer(10.0),
			},
		})
		collection.Schema.AddField(&schema.SchemaField{
			Id:   "f3_id",
			Name: "f3_name",
			Type: schema.FieldTypeBool,
		})

		// use different dao to avoid triggering automigrate while saving the dummy collection
		if err := daos.New(app.DB()).SaveCollection(collection); err != nil {
			t.Fatalf("[%d] Failed to save dummy collection, got %v", i, err)
		}

		// @todo remove after collections cache is replaced
		app.Bootstrap()

		collection.Name = "test456_update"
		collection.Type = models.CollectionTypeBase
		collection.DeleteRule = types.Pointer(`updated > 0 && @request.auth.id != ''`)
		collection.ListRule = nil
		collection.NormalizeOptions()
		collection.Schema.RemoveField("f3_id")
		collection.Schema.AddField(&schema.SchemaField{
			Id:   "f4_id",
			Name: "f4_name",
			Type: schema.FieldTypeText,
			Options: &schema.TextOptions{
				Pattern: "`test backtick`123",
			},
		})
		f := collection.Schema.GetFieldById("f2_id")
		f.Name = "f2_name_new"

		// save the changes and trigger automigrate
		if err := app.Dao().SaveCollection(collection); err != nil {
			t.Fatalf("[%d] Failed to save dummy collection changes, got %v", i, err)
		}

		files, err := os.ReadDir(migrationsDir)
		if err != nil {
			t.Fatalf("[%d] Expected migrationsDir to be created, got: %v", i, err)
		}

		if total := len(files); total != 1 {
			t.Fatalf("[%d] Expected 1 file to be generated, got %d", i, total)
		}

		expectedName := "_updated_test456." + s.lang
		if !strings.Contains(files[0].Name(), expectedName) {
			t.Fatalf("[%d] Expected filename to contains %q, got %q", i, expectedName, files[0].Name())
		}

		fullPath := filepath.Join(migrationsDir, files[0].Name())
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("[%d] Failed to read the generated migration file: %v", i, err)
		}

		if v := strings.TrimSpace(string(content)); v != strings.TrimSpace(s.expectedTemplate) {
			t.Fatalf("[%d] Expected template \n%v \ngot \n%v", i, s.expectedTemplate, v)
		}
	}
}

func TestAutomigrateCollectionNoChanges(t *testing.T) {
	scenarios := []struct {
		lang string
	}{
		{
			migratecmd.TemplateLangJS,
		},
		{
			migratecmd.TemplateLangGo,
		},
	}

	for i, s := range scenarios {
		app, _ := tests.NewTestApp()
		defer app.Cleanup()

		migrationsDir := filepath.Join(app.DataDir(), "_test_migrations")

		migratecmd.MustRegister(app, nil, &migratecmd.Options{
			TemplateLang: s.lang,
			Automigrate:  true,
			Dir:          migrationsDir,
		})

		// create dummy collection
		collection := &models.Collection{}
		collection.Name = "test123"
		collection.Type = models.CollectionTypeAuth

		// use different dao to avoid triggering automigrate while saving the dummy collection
		if err := daos.New(app.DB()).SaveCollection(collection); err != nil {
			t.Fatalf("[%d] Failed to save dummy collection, got %v", i, err)
		}

		// @todo remove after collections cache is replaced
		app.Bootstrap()

		// resave without changes and trigger automigrate
		if err := app.Dao().SaveCollection(collection); err != nil {
			t.Fatalf("[%d] Failed to save dummy collection update, got %v", i, err)
		}

		files, _ := os.ReadDir(migrationsDir)
		if total := len(files); total != 0 {
			t.Fatalf("[%d] Expected 0 files to be generated, got %d", i, total)
		}
	}
}
