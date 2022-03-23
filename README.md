# PHP Start Cloud Native Buildpack
## `gcr.io/paketo-buildpacks/php-start`

A Cloud Native Buildpack for running HTTPD, Nginx, and/or FPM start commands
for PHP apps.

## Behavior

This buildpack will always participate if it's `requirements` are met. In the
HTTPD server case `requires` `php`, `php-fpm` optionally, `httpd`, and
`httpd-config`. In the Nginx case, it will require `nginx` and `nginx-config`
instead of `httpd` and `httpd-config`. These requirements will be met when used
in conjunction with the other buildpacks in the Paketo PHP language family.

| Requirement        | Build | Launch |
|--------------------|-------|--------|
| `php`              | x     |        |
| `php-fpm`          | x     | x      |
| `httpd` or `nginx` | x     |        |
| `httpd-config` or `nginx-config` | x     | x      |


It will set the default start command to something that looks like:
```
procmgr-binary /layers/paketo-buildpacks_php-start/php-start/procs.yml
```

The `procmgr-binary` is a process manager that will run multiple start commands
on the container. This is done to allow for FPM to run on the container
alongside the web server. The `procs.yml` file it runs with contains the
commands and arguments for both `php-fpm` and the web-server.

When the buildpack runs, you will see in the logs what processes are addded to
procs.yml.


## Integration

This CNB writes a start command, so there's currently no scenario we can
imagine that you would need to require it as dependency.

## Usage

To package this buildpack for consumption:

```
$ ./scripts/package.sh --version <version-number>
```

This will create a `buildpackage.cnb` file under the `build` directory which you
can use to build your app as follows:
```
pack build <app-name> -p <path-to-app> -b build/buildpackage.cnb
```

## Run Tests

To run all unit tests, run:
```
./scripts/unit.sh
```

To run all integration tests, run:
```
/scripts/integration.sh
```

## Debug Logs
For extra debug logs from the image build process, set the `$BP_LOG_LEVEL`
environment variable to `DEBUG` at build-time (ex. `pack build my-app --env
BP_LOG_LEVEL=DEBUG` or through a  [`project.toml`
file](https://github.com/buildpacks/spec/blob/main/extensions/project-descriptor.md).
