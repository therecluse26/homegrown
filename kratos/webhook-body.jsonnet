// Minimal passthrough webhook body template.
// The IAM domain (01-iam) will extend this to extract specific identity traits.
// See: kratos/kratos.yml hooks configuration
function(ctx) {
  identity_id: ctx.identity.id,
  schema_id: ctx.identity.schema_id,
  traits: ctx.identity.traits,
}
