# Sample Challenges

Below are some of the sample challenges wrapped in the format that beast can understand and deploy.

## Service Challenge

These type of challenges need not only consist of binary challenges any executable can be hosted using service type
challenge which includes scripts too. Shell scripts, ruby scripts, python scripts can all be hosted using this type, make
sure you include the proper shebang at the top of script if you are hosting scripts
For binary service type challenges the author needs to build the binary first, this can be done using the setup scripts.

```toml
[author]
name = "fristonio"
email = "deepeshpathak09@gmail.com"
ssh_key = "ssh-rsa AAAAB3NzaC1y"

[challenge.metadata]
name = "sample"
flag = "CTF{sample_flag}"
type = "service"
points = 40

[challenge.env]
apt_deps = ["gcc", "socat"]
setup_scripts = ["setup.sh"]
service_path = "pwn"
ports = [10001]
```

Setup script

```bash
set -e

gcc -o pwn pwn_me.c

exit 0
```

## Docker Challenge

To further improve the customizability/flexibility of challenge deployment a docker type challenge is provided. This can
be used in cases when the author knows how to create docker images, this brings to table the customization of the entire
container environment that beast can deploy.

Any existing dockerized challenge can be easily ported to beast by simply creating the `beast.toml` configuration file with only
a few required fields.

```toml
[author]
name = "fristonio"
email = "deepeshpathak09@gmail.com"
ssh_key = "ssh-rsa AAAAB3NzaC1y"

[challenge.metadata]
name = "docker-type"
flag = "CTF{sample_flag}"
type = "docker"
points = 50

[challenge.env]
docker_context = "Dockerfile"
ports = [10002]
```

```Dockerfile
FROM ubuntu:16.04

RUN apt-get update \
	&& apt-get install -y gcc socat

COPY script.sh /script.sh
COPY entrypoint.sh /entry.sh
COPY flag /flag

EXPOSE 10002

RUN chmod +x /script.sh
RUN chmod +x /entry.sh

CMD ["/entrypoint.sh"]
```

## Static challenges

Some challenge don't need a complete environment to run, they just need to serve some static files to the user. This
happens a lot in case of forensics challenges. Beast optimizes the deployment of such challenges by creating just a single
environment(nginx based file server) for all such challenges.

```toml
[author]
name = "fristonio"
email = "deepeshpathak09@gmail.com"
ssh_key = "ssh-rsa AAAAB3NzaC1y"

[challenge.metadata]
name = "static-challenge"
flag = "CTF{sample_flag}"
type = "static"
points = 100
```

In the above case all the files that are present in the challenge root are available for the player to download.

### Note

Every challenge in all the above provided types have a way to provide static files for the user to download, this can be
done using the `static_dir` in the challenge configuration. Every file in the provided `static_dir` directory will be provided
for download as a static asset.

## Bare beast Challenge

For these type challenges the key is the `run_cmd` configuration parameter in the `beast.toml` file. This
consist of the command to run for the container, any challenge can be deployed using this type which can
be customized by the using the corresponding `run_cmd`.

For example a django challenge can be deployed by poviding `python manage.py` in the `run_cmd`.

```toml
[author]
name = "fristonio"
email = "deepeshpathak09@gmail.com"
ssh_key = "ssh-rsa AAAAB3NzaC1y"

[challenge.metadata]
name = "sample"
flag = "CTF{sample_flag}"
type = "bare"
hints = ["simple_hint_1", "simple_hint_2"]

[challenge.env]
apt_deps = ["gcc", "socat"]
setup_scripts = ["setup.sh"]
run_cmd = "socat tcp-l:10003,fork,reuseaddr exec:./pwn"
ports = [10003]
```

Setup script

```bash
set -e

gcc -o pwn pwn_me.c

exit 0
```

## Web Challenges

Beast can be used to deploy different type of challenges when in comes to web category. Currently beast supports the following type of web challenges 

* PHP
* Node
* Python - Django and Flask

The type and environment of the challenge is decided by `type` field in the `challenge.metadata` section. The format of which is - 
`web:<Language>:<Version>:<language-extension>`

### Sample PHP challenge

```toml
[author]
name = "fristonio"
email = "deepeshpathak09@gmail.com"
ssh_key = "ssh-rsa AAAAB3NzaC1y"

[challenge.metadata]
name = "web-php"
flag = "CTF{sample_flag}"
type = "web:php:7.1:cli"

[challenge.env]
ports = [10002]
web_root = "challenge"
default_port = 10002
```

The `web_root` is the base directory for the php server to locate the files.

The type of challenge consist of the following format - `web:php:<PHP Version>:<cil/apache>`

### PHP challenge with MySQL database

For deploying a challenge with database requirement beast sidecars needs to be used.

```
[author]
name = "fristonio"
email = "deepeshpathak09@gmail.com"
ssh_key = "ssh-rsa AAAAB3NzaC1y"

[challenge.metadata]
name = "web-php-mysql"
flag = "CTF{sample_flag}"
type = "web:php:7.1:cli"
sidecar = "mysql"

[challenge.env]
apt_deps = ["gcc", "php*-mysql"]
setup_scripts = ["setup.sh"]
ports = [10004]
web_root = "challenge"
default_port = 10004
```

The above configuration will create a new database in the globally present MySQL instance and will put the connection
credentials in the Environment variables. For MySQL database these credentials are

* MYSQL_database
* MYSQL_username
* MYSQL_password

These environment variables can then be used to connect to the database and perform the required action.

```php
<?php 
$dsn = "mysql:host=mysql;dbname=" . getenv("MYSQL_database") . ";charset=utf8mb4";
$options = [
  PDO::ATTR_EMULATE_PREPARES   => false,
  PDO::ATTR_ERRMODE            => PDO::ERRMODE_EXCEPTION,
  PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
];

try {
	$pdo = new PDO($dsn, getenv("MYSQL_username"), getenv("MYSQL_password"), $options);
} catch (Exception $e) {
	error_log($e->getMessage());
	echo $e->getMessage();
	exit('Something weird happened');
}

echo "Success: A proper connection to MySQL was made! The my_db database is great." . PHP_EOL;

?>
```

For other databases, more information can be found on Sidecars documentation.

#### Note

Make sure that you are installing all the mysql related dependencies in the `apt_deps` configuration parameter. As in the 
above case `php*-mysql`
