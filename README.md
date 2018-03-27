# Swagger Generate Types

Generate Go types for definitions and responses in a Swagger V2 spec.

* Use `x-nullable: true` to make something a pointer, defaults to false
* Use `required` on the object to make a field not `omitempty`, defaults to
  `omitempty`.
* Use `title` of a schema to set the Go struct of field name.
