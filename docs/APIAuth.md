# Authentication

Authentication is done to provide restricted access only to the organizers. Authentication is done using asymmetric encryption. After authentication, JWT tokens are used to provide access for further usage of API's.

## Usage

### Configuration

For signing JWT tokens using **HMAC** algorithm `jwt_secret` is required which can be configured in the beast global config (config.toml) as:

```toml
jwt_secret = "beast_jwt_secret_SUPER_STRONG_0x100010000100"
```

The SSH keys are used for assymetric authentication which can be registered directly from the terminal using:

`beast create-author --name <username> --email <email> --publickey <pub-key-location>`

Or it can be given in the challenge description(beast.toml) : 

```toml
[author]
ssh_key = "<public key>"
```

This key gets added in the database when beast.toml is validated.

## Flow

Once the public key is registered in the database, the user can get a JWT token through the following steps :
* First make a GET request on URL : `/auth/<username>`
* The response will be of the format :

``` JSON
{
    "challenge"	:	"Challenge String",
    "message"	:	"<solve message>"
}

```

* The challenge must be decrypted using *ssh private key* and then a POST request has to be made on URL: `/auth/<username>`along with POST form data: `decrmess=<decrypted message>`
* You will get a response like this : 
``` JSON
{
    "token"    :	"YOUR_AUTHENTICATION_TOKEN",
    "message"  :	"<Usage message>"
}

```

* Now to access any restricted route you need to add this JWT token in the HTTP header as :
``` HTTP
Authorization: Bearer YOUR_AUTHENTICATION_TOKEN
```

### Alternative

If you have the `beast` binary with you, you can also use command :

`beast getauth --identity <path to ssh-private-key> --username <username> --host <host-string>`

This command will give you the JWT token for usage in other APIs by adding in the HTTP header.