-- +goose Up
CREATE TABLE departments (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(200) NOT NULL,
    parent_id  INTEGER REFERENCES departments(id) ON DELETE RESTRICT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- уникальность имени в пределах одного parent (NULL parent считается отдельной группой)
CREATE UNIQUE INDEX idx_dept_name_parent
    ON departments (name, COALESCE(parent_id, 0));

CREATE TABLE employees (
    id            SERIAL PRIMARY KEY,
    department_id INTEGER   NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    full_name     VARCHAR(200) NOT NULL,
    position      VARCHAR(200) NOT NULL,
    hired_at      DATE,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS departments;
