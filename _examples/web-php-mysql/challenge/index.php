<?php 

$dsn = "mysql:host=mysql;dbname=" . getenv("MYSQL_database") . ";charset=utf8mb4";
$options = [
  PDO::ATTR_EMULATE_PREPARES   => false, // turn off emulation mode for "real" prepared statements
  PDO::ATTR_ERRMODE            => PDO::ERRMODE_EXCEPTION, //turn on errors in the form of exceptions
  PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC, //make the default fetch be an associative array
];

try {
	$pdo = new PDO($dsn, getenv("MYSQL_username"), getenv("MYSQL_password"), $options);
} catch (Exception $e) {
	error_log($e->getMessage());
	echo $e->getMessage();
	exit('Something weird happened'); //something a user can understand
}

echo "Success: A proper connection to MySQL was made! The my_db database is great." . PHP_EOL;

?>