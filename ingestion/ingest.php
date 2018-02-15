<?php

require_once "vendor/autoload.php";

header("Content-Type: text/plain");

$queueName = "queue:postback";

$body = file_get_contents('php://input');
$json = json_decode($body);

$redis = new Predis\Client([
    "host" => "redis"
]);
$redis->rpush($queueName, json_encode($json));

http_response_code(204);

?>
