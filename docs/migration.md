# Database Migration Guide

## Overview

The tasked planner uses SQLite with automatic schema migration. When new features are added that require database schema changes, the migration happens automatically when opening an existing database.

## Migration Process

### Automatic Migration

The planner uses a robust migration strategy:

1. **Schema Definition**: All database schema is defined in `planner/schema.sql`
2. **Safe Creation**: All tables, indexes, and triggers use `CREATE ... IF NOT EXISTS` clauses
3. **Automatic Execution**: When creating a new planner instance, the full schema is executed
4. **Backward Compatibility**: Existing data is preserved during schema updates

### Migration for References Feature

The references feature (added in version X.X.X) introduced a new `step_references` table to store reference URLs for plan steps.

#### What Happens During Migration

When opening an existing database that doesn't have the `step_references` table:

1. The planner initialization automatically executes `schema.sql`
2. The `step_references` table is created with `CREATE TABLE IF NOT EXISTS`
3. All existing tables remain unchanged
4. Existing data (plans, steps, acceptance criteria) is preserved
5. New functionality becomes available immediately

#### Data Integrity

- **No Data Loss**: All existing plans and steps remain intact
- **No Downtime**: Migration happens instantly during database open
- **No Manual Steps**: Migration is completely automatic
- **Rollback Safe**: If needed, you can revert to an older version (references will simply be ignored)

## Testing Migration

The migration process is thoroughly tested in `planner/migration_test.go`:

### Test Coverage

1. **Schema Validation**: Ensures all DDL statements use `IF NOT EXISTS`
2. **Old Database Simulation**: Creates a database without the new table
3. **Migration Verification**: Confirms the new table is created automatically
4. **Data Preservation**: Verifies all existing data remains intact
5. **Functionality Testing**: Confirms existing operations still work
6. **New Feature Integration**: Tests that new features work with migrated data
7. **Backward Compatibility**: Ensures code that doesn't use new features continues to work

### Running Migration Tests

```bash
# Run migration-specific tests
go test -v ./planner -run TestDatabaseMigration

# Run schema validation tests
go test -v ./planner -run TestSchemaValidation

# Run all planner tests to ensure compatibility
go test -v ./planner
```

## Migration Strategy

### Current Strategy: Schema Recreation

The current migration strategy re-executes the entire schema on every database open:

**Advantages:**
- Simple and reliable
- Automatically handles any schema changes
- No version tracking needed
- Self-healing (corrupted schema gets fixed)

**Considerations:**
- Minimal performance overhead (schema operations are fast)
- Safe due to `IF NOT EXISTS` clauses
- Works well for SQLite with modest schema complexity

### Future Considerations

For more complex migration scenarios, consider:

- **Migration Scripts**: Separate migration files for complex transformations
- **Version Tracking**: Database version metadata for conditional migrations  
- **Data Transformations**: Scripts for restructuring existing data
- **Rollback Support**: Downgrade migrations for version rollbacks

Currently, the simple schema recreation approach is sufficient for the planner's needs.

## Best Practices

### For Developers

When adding new database schema elements:

1. **Always use `IF NOT EXISTS`** in `schema.sql`
2. **Add migration tests** for the new feature
3. **Verify backward compatibility** with existing code
4. **Test with real database** files from previous versions
5. **Document breaking changes** if any

### For Users

- **Backup Important Data**: While migration is safe, backups are always recommended
- **Test in Development**: Try new versions with copies of your database first
- **No Manual Steps Required**: Migration happens automatically

## Troubleshooting

### Common Issues

1. **Permission Errors**: Ensure write access to the database file and directory
2. **Disk Space**: Verify sufficient space for temporary migration operations
3. **File Locks**: Close all connections to the database before upgrading
4. **Schema Conflicts**: Manually corrupted schema may need manual intervention

### Recovery

If migration fails:

1. **Check Error Messages**: Usually indicate permission or disk space issues
2. **Restore from Backup**: Use a known-good backup if available
3. **Manual Schema Fix**: Advanced users can manually fix schema issues
4. **Start Fresh**: Create a new database and manually recreate plans if needed

### Support

For migration issues:

1. Check the test cases in `migration_test.go` for expected behavior
2. Run tests with `-v` flag for detailed output
3. Examine the `schema.sql` file for the expected final state
4. File an issue with error messages and steps to reproduce

## Schema Evolution History

### v1.0.0 - Initial Schema
- `plans` table
- `steps` table  
- `step_acceptance_criteria` table

### v1.1.0 - References Feature
- Added `step_references` table
- Added indexes and triggers for references
- Backward compatible migration

*Note: Actual version numbers may differ*
