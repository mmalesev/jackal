# jackal

An XMPP server written in Go.

This repository is a fork of [ortuman/jackal](https://github.com/ortuman/jackal) making it available for SCION/QUIC. Refer to the original repository for general usage.

If you have go1.12 installed (not supporting go1.11 at the moment), you can build jackal using Makefile.
```shell
make build
```

You can check if the project has built successfully by running the following command. You should see jackal logo, together with the usage instructions.
```shell
./jackal -h
```
## Running jackal
In order to run jackal, you have to specify the configuration in .yml file. An example .yml file is provided in the repository as example.jackal.yml. You need to do the following steps before you can run the server with the configuration specified in the example.jackal.yml.

### SCION config
In .yml configuration file, you can provide different options for the incoming SCION connections:
* addr: SCION address where the server is listening for incoming connections (if "localhost", jackal determines the SCION localhost on startup)
* port: Port listening for incoming SCION connections (default is 52690)
* keep_alive: Time after which the unresponsive connection is broken.
* cert_path: Absolute path to the certificate used in creating QUIC connection (can be the same as the certificate used in c2s IP connection)
* priv_key_path: Private key corresponding to the certificate, also used in creating QUIC connection

### MySQL database creation
Install MySQL:

```shell
sudo apt-get install mysql-server
mysql_secure_installation
```
Grant right to a dedicated 'jackal' user (replace `password` with your desired password).

```shell
echo "GRANT ALL ON jackal.* TO 'jackal'@'localhost' IDENTIFIED BY 'password';" | mysql -h localhost -u root -p
```

Create 'jackal' database (using previously created password).

```shell
echo "CREATE DATABASE jackal;" | mysql -h localhost -u jackal -p
```

Download lastest version of the [MySQL schema](sql/mysql.up.sql) from jackal Github repository.

```shell
wget https://raw.githubusercontent.com/ortuman/jackal/master/sql/mysql.up.sql
```

Load database schema into the database.

```shell
mysql -h localhost -D jackal -u jackal -p < mysql.up.sql
```

Your database is now ready to connect with jackal.

### Name resolution
In the router/hosts section of the .yml file, replace the "name" entry with your server name. Make sure that this name resolves over DNS to a valid IP address where the server will be accepting client requests. Also, specify the paths to the TLS private key and certificate in the tls section, as well as in the scion_transport section.
Finally, your hostname specified in the router/hosts section needs to resolve to a valid SCION addres on the specified [RAINS](https://github.com/netsec-ethz/rains) server. SCION address where the RAINS server is running needs to be specified in the config file at ~/go/src/github.com/scionproto/scion/gen/rains.cfg. Simply put the address of the RAINS server together with the port inside this file.
NOTE: when building jackal, both SCION and IP address mapings from /etc/hosts file will be loaded.

### Starting jackal
Once you have the config file setup correctly, together with the MySQL database, valid certificates and the DNS and RAINS entries, you can run your jackal server. 
```shell
./jackal -c example.jackal.yml
```
## Connecting to jackal
In order to connect to an XMPP server, you can use an XMPP client, for example [Psi](https://psi-im.org/) (GUI, has in-band registration) or [Profanity](https://profanity-im.github.io/) (console-based, does not support in-band registration). The following are the steps if using PSI:
1. General->Account Setup->Add
2. Pick a name, check the "Register new account" box
3. Add your XMPP server name, e.g. myjackalserver.com (this name should be DNS/RAINS resolvable) - if you are using a self-signed certificate you will need to confirm the server's authenticity by clicking "Trust this certificate"
4. Provide your desired username and password
Now you should have a new account registered at your xmpp server. You can set the status of the account to online, and send/receive messages to other xmpp clients.

## Notes for local testing
To be used only when playing with jackal.

### User registration
Since some XMPP clients do not support in-band registration (e.g. Profanity), users need to be created manually. So far, the only way is to manually add them into the MySQL database created previously. For example:

```shell
mysql -h localhost -u jackal -p
use jackal;
insert into users (`username`, `password`, `last_presence`, `last_presence_at`, `updated_at`, `created_at`) values ('user1', 'asdf', '<presence from="user1@localhost/profanity" to="user1@localhost" type="unavailable"/>', '2019-04-19 18:42:58', '2019-04-19 18:42:58', '2019-04-19 18:42:58');

```

### Generating self-signed certificates
If you need to create self-signed certificates, you might find this [post](https://stackoverflow.com/questions/21488845/how-can-i-generate-a-self-signed-certificate-with-subjectaltname-using-openssl) useful.
