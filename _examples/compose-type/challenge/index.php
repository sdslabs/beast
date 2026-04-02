<?php

$host = getenv('MYSQL_HOST') ?: 'mysql';
$db = getenv('MYSQL_DATABASE') ?: 'my_db';
$user = getenv('MYSQL_USER') ?: 'my_user';
$pass = getenv('MYSQL_PASSWORD') ?: 'my_password';

$dsn = "mysql:host=$host;dbname=$db;charset=utf8mb4";
$options = [
  PDO::ATTR_EMULATE_PREPARES => false,
  PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
  PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
];

try {
  $pdo = new PDO($dsn, $user, $pass, $options);
  echo "OK: connected to MySQL\n";
} catch (Exception $e) {
  http_response_code(500);
  echo "ERROR: " . $e->getMessage() . "\n";
}

