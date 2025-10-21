// Package configurator provides utilities for resolving configuration values from files
// Usage: Add `_file` suffix fields to your config structs and call ResolveFiles()
package configurator

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const TAG_NAME = "fileconfig"
const SKIP_COMMAND = "skip"
const FILE_POSTFIX = "File"

// ReadSecretFile reads a secret from a file, trimming whitespace
func ReadSecretFile(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w", path, err)
	}

	// Trim whitespace and newlines that are common in secret files
	return strings.TrimSpace(string(data)), nil
}

// GetStringOrFile gets a value from viper, checking for a _file variant first
// Example: GetStringOrFile(conf, "password") checks "password_file" first, then "password"
func GetStringOrFile(conf *viper.Viper, key string) (string, error) {
	fileKey := key + "_file"
	if filePath := conf.GetString(fileKey); filePath != "" {
		value, err := ReadSecretFile(filePath)
		if err != nil {
			return "", fmt.Errorf("loading %s from file: %w", key, err)
		}
		if value != "" {
			log.Debug().
				Str("key", key).
				Str("file", filePath).
				Msg("loaded value from file")
			return value, nil
		}
	}

	return conf.GetString(key), nil
}

// ResolveFiles walks through a struct and resolves any *_file fields into their corresponding value fields
// It uses reflection to find fields ending in "File" and resolves them to their base field
//
// Example:
//
//	type Config struct {
//	    Password     string `mapstructure:"password"`
//	    PasswordFile string `mapstructure:"password_file"`
//	}
//	// After ResolveFiles, if PasswordFile is set, Password will contain the file contents
func ResolveFiles(config any) error {
	return resolveFilesRecursive(reflect.ValueOf(config), "")
}

func resolveFilesRecursive(v reflect.Value, path string) error {
	// dereference pointers
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	// only process structs
	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()

	// build map of file fields and also which fields keep as a path
	fileFields := make(map[string]fileFieldInfo)
	skipFields := make(map[int]bool)

	// Build a map of file fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Name

		if tag := field.Tag.Get(TAG_NAME); tag == SKIP_COMMAND {
			skipFields[i] = true
			continue
		}

		// Check if this is a *File field
		if strings.HasSuffix(fieldName, FILE_POSTFIX) && field.Type.Kind() == reflect.String {
			baseName := strings.TrimSuffix(fieldName, FILE_POSTFIX)
			fileFields[baseName] = fileFieldInfo{
				fileFieldIndex: i,
				filePath:       v.Field(i).String(),
			}
		}
	}

	// Now iterate again to resolve file fields into their base fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Name
		fieldValue := v.Field(i)

		if skipFields[i] {
			continue
		}

		if fileInfo, hasFileField := fileFields[fieldName]; hasFileField {
			if fileInfo.filePath != "" && field.Type.Kind() == reflect.String {
				// Read the file and set the value
				content, err := ReadSecretFile(fileInfo.filePath)
				if err != nil {
					fieldPath := path
					if fieldPath != "" {
						fieldPath += "."
					}
					fieldPath += fieldName
					return fmt.Errorf("resolving %s: %w", fieldPath, err)
				}

				if content != "" && fieldValue.CanSet() {
					fieldValue.SetString(content)
					log.Trace().
						Str("field", fieldName).
						Str("file", fileInfo.filePath).
						Msg("resolved field from file")
				}
			}
		}

		// Recursively process nested structs
		if field.Type.Kind() == reflect.Struct ||
			(field.Type.Kind() == reflect.Pointer && field.Type.Elem().Kind() == reflect.Struct) {
			newPath := path
			if newPath != "" {
				newPath += "."
			}
			newPath += fieldName

			if err := resolveFilesRecursive(fieldValue, newPath); err != nil {
				return err
			}
		}

		// Process maps of structs (like Providers map)
		if field.Type.Kind() == reflect.Map &&
			(field.Type.Elem().Kind() == reflect.Struct ||
				(field.Type.Elem().Kind() == reflect.Pointer && field.Type.Elem().Elem().Kind() == reflect.Struct)) {
			if !fieldValue.IsNil() {
				iter := fieldValue.MapRange()
				for iter.Next() {
					mapValue := iter.Value()

					// For map values, we need to create a new value since map values aren't addressable
					if mapValue.Kind() == reflect.Struct {
						// Create a new addressable copy
						newValue := reflect.New(mapValue.Type()).Elem()
						newValue.Set(mapValue)

						if err := resolveFilesRecursive(newValue, path+"."+fieldName+"["+iter.Key().String()+"]"); err != nil {
							return err
						}

						// Set the modified value back into the map
						fieldValue.SetMapIndex(iter.Key(), newValue)
					} else {
						if err := resolveFilesRecursive(mapValue, path+"."+fieldName+"["+iter.Key().String()+"]"); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	return nil
}

type fileFieldInfo struct {
	fileFieldIndex int
	filePath       string
}

// UnmarshalWithFileResolution is a convenience function that unmarshals config and resolves files in one call
func UnmarshalWithFileResolution(conf *viper.Viper, target any) error {
	if err := conf.Unmarshal(target); err != nil {
		return fmt.Errorf("unmarshaling config: %w", err)
	}

	if err := ResolveFiles(target); err != nil {
		return fmt.Errorf("resolving file references: %w", err)
	}

	return nil
}
