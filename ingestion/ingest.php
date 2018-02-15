<?php

require_once "vendor/autoload.php";

header("Content-Type: text/plain");

$redisHost = getenv("REDIS_URL") ?: "redis://localhost:6379";
$queueName = "queue:postback";

$body = file_get_contents('php://input');
$json = json_decode($body);

$redis = new Predis\Client($redisHost);
$redis->rpush($queueName, json_encode($json));

http_response_code(204);

?>
