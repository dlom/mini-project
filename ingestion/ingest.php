<?php

require_once "vendor/autoload.php";

header("Content-Type: text/plain");

$redisHost = getenv("REDIS_URL") ?: "redis://localhost:6379";
$queueName = getenv("REDIS_QUEUE") ?: "queue:postback";

// Ensure that the request isn't malformed or malicious
if ($_SERVER["REQUEST_METHOD"] !== "POST") {
    http_response_code(405);
    exit("405 Method Not Allowed");
}

if ($_SERVER["CONTENT_TYPE"] !== "application/json") {
    http_response_code(415);
    exit("400 Bad Request");
}

// Grab the raw body from the POST request
$body = file_get_contents("php://input");
if ($body === false) {
    http_response_code(500);
    exit("500 Internal Server Error");
}

// Decode the body from its JSON representation
$json = json_decode($body);
if ($json === NULL) {
    http_response_code(400);
    exit("400 Bad Request");
}

// Perform simple validation on the JSON
$validator = new JsonSchema\Validator();
$validator->validate($json, [
    '$ref' => "file://" . realpath("schema.json")
]);
if (!$validator->isValid()) {
    http_response_code(422);
    exit("422 Unprocessable Entity");
}

// Push the encoded JSON to redis
$redis = new Predis\Client($redisHost);
$redis->rpush($queueName, json_encode($json));

http_response_code(204);

?>
