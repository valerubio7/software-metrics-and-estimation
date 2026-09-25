# Delta for Project

## ADDED Requirements

### Requirement: Replace the basic data of an existing project

The system MUST provide `PUT /projects/{project_id}` to replace the complete basic-data representation of the identified project. A valid request MUST provide `name`, `start_date`, and `planned_finish_date`. On success, the system MUST persist the supplied values and return the updated project's identifier and basic data.

#### Scenario: Update all basic fields of an existing project

- GIVEN an existing project and a request containing a nonblank `name`, a valid `start_date`, and a valid `planned_finish_date` that is not before the start date
- WHEN the client sends `PUT /projects/{project_id}` with the project's ID
- THEN the system persists all three supplied basic fields for that project
- AND returns a successful response containing the same project ID and the updated basic fields

#### Scenario: Accept a planned finish date equal to the start date

- GIVEN an existing project and a complete request whose `planned_finish_date` equals its `start_date`
- WHEN the client sends `PUT /projects/{project_id}`
- THEN the system accepts and persists the updated basic data

### Requirement: Reject incomplete or invalid project update data

The system MUST reject an update request if any required basic field is missing or invalid, MUST explain the validation reason, and MUST NOT persist any part of the rejected update.

#### Scenario: Reject a request missing a required basic field

- GIVEN an existing project
- AND an update request omits `name`, `start_date`, or `planned_finish_date`
- WHEN the client sends `PUT /projects/{project_id}`
- THEN the system returns a validation error identifying the missing field
- AND the project's previously stored basic data remains unchanged

#### Scenario: Reject a blank project name

- GIVEN an existing project and an update request with a missing or blank `name`
- WHEN the client sends `PUT /projects/{project_id}`
- THEN the system returns a validation error identifying the invalid name
- AND the project's previously stored basic data remains unchanged

#### Scenario: Reject an invalid date value

- GIVEN an existing project and a complete update request with an invalid `start_date` or `planned_finish_date`
- WHEN the client sends `PUT /projects/{project_id}`
- THEN the system returns a validation error identifying the invalid date
- AND the project's previously stored basic data remains unchanged

### Requirement: Enforce project date consistency during updates

The system MUST reject an update when `planned_finish_date` is earlier than `start_date`, MUST explain the date inconsistency, and MUST NOT persist any part of the rejected update.

#### Scenario: Reject a planned finish date before the start date

- GIVEN an existing project and a complete update request whose `planned_finish_date` is earlier than `start_date`
- WHEN the client sends `PUT /projects/{project_id}`
- THEN the system returns a validation error explaining that the planned finish date cannot be earlier than the start date
- AND the project's previously stored basic data remains unchanged

### Requirement: Report an unknown project during update

The system MUST return a not-found response when the ID in `PUT /projects/{project_id}` does not identify an existing project, and MUST NOT create a project or alter any existing project data.

#### Scenario: Update a project with an unknown ID

- GIVEN the requested project ID does not exist
- WHEN the client sends `PUT /projects/{project_id}` with otherwise valid basic data
- THEN the system returns a not-found response
- AND no project is created and no existing project data is changed

### Requirement: Preserve data outside the project's basic fields

When updating a project, the system MUST change only that project's `name`, `start_date`, and `planned_finish_date`. It MUST preserve the project's identifier, other project data, and all member and story data associated with or otherwise unrelated to the project.

#### Scenario: Update basic data without changing unrelated data

- GIVEN an existing project with an identifier, other project data, associated members, and associated stories
- WHEN the client successfully updates the project's `name`, `start_date`, and `planned_finish_date`
- THEN the project retains the same identifier and all other project data
- AND all associated member and story data remains unchanged
