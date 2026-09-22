## Testing

Ensure Go and Terraform are installed.

Run tests by specifying the example parameter when running make test. Replace default with the desired example from the examples directory.

Add skip-destroy=true to skip the terraform destroy step, like make test example=default skip-destroy=true.

Run all tests in parallel or sequentially with make test-parallel or make test-sequential. Exclude specific examples by adding exception=example1,example2, where example1 and example2 are examples to skip.

These tests verify the module's reliability across various configurations.
