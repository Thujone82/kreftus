<?php
declare(strict_types=1);

header('Content-Type: application/json; charset=utf-8');
header('Cache-Control: no-store, no-cache, must-revalidate');

function respondJsonError(int $status, string $code, string $message): never {
    http_response_code($status);
    echo json_encode([
        'error' => [
            'code' => $code,
            'message' => $message,
        ],
    ], JSON_UNESCAPED_SLASHES);
    exit;
}

function getAllowedHosts(): array {
    $hosts = ['kreft.us', 'www.kreft.us', 'localhost', '127.0.0.1'];
    $httpHost = $_SERVER['HTTP_HOST'] ?? '';
    if ($httpHost !== '') {
        $hostOnly = strtolower((string) preg_replace('/:\d+$/', '', $httpHost));
        if ($hostOnly !== '' && !in_array($hostOnly, $hosts, true)) {
            $hosts[] = $hostOnly;
        }
    }
    return $hosts;
}

function isAllowedHost(?string $url, array $allowedHosts): bool {
    if ($url === null || $url === '') {
        return false;
    }
    $host = parse_url($url, PHP_URL_HOST);
    if (!is_string($host) || $host === '') {
        return false;
    }
    return in_array(strtolower($host), $allowedHosts, true);
}

$allowedHosts = getAllowedHosts();
$origin = $_SERVER['HTTP_ORIGIN'] ?? null;
$referer = $_SERVER['HTTP_REFERER'] ?? null;

if (is_string($origin) && $origin !== '') {
    if (!isAllowedHost($origin, $allowedHosts)) {
        respondJsonError(403, 'FORBIDDEN_ORIGIN', 'Origin not allowed');
    }
    header('Access-Control-Allow-Origin: ' . $origin);
    header('Vary: Origin');
    if (($_SERVER['REQUEST_METHOD'] ?? '') === 'OPTIONS') {
        header('Access-Control-Allow-Methods: GET, POST, OPTIONS');
        header('Access-Control-Allow-Headers: Content-Type, API-KEY');
        http_response_code(204);
        exit;
    }
} elseif (is_string($referer) && $referer !== '') {
    if (!isAllowedHost($referer, $allowedHosts)) {
        respondJsonError(403, 'FORBIDDEN_ORIGIN', 'Origin not allowed');
    }
}

function extractApiKey(): string {
    if (isset($_SERVER['HTTP_API_KEY']) && trim((string) $_SERVER['HTTP_API_KEY']) !== '') {
        return trim((string) $_SERVER['HTTP_API_KEY']);
    }
    if (function_exists('getallheaders')) {
        $headers = getallheaders();
        if (is_array($headers)) {
            foreach ($headers as $name => $value) {
                if (strtolower((string) $name) === 'api-key' && trim((string) $value) !== '') {
                    return trim((string) $value);
                }
            }
        }
    }
    return '';
}

function forwardRequest(string $method, string $url, array $headers, ?string $body, int $timeoutSeconds): array {
    if (function_exists('curl_init')) {
        $ch = curl_init($url);
        if ($ch === false) {
            return [502, json_encode(['error' => ['code' => 'UPSTREAM_ERROR', 'message' => 'Failed to initialize cURL']])];
        }
        $opts = [
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_CUSTOMREQUEST => $method,
            CURLOPT_HTTPHEADER => $headers,
            CURLOPT_TIMEOUT => $timeoutSeconds,
            CURLOPT_CONNECTTIMEOUT => 15,
            CURLOPT_SSL_VERIFYPEER => true,
        ];
        if ($body !== null) {
            $opts[CURLOPT_POSTFIELDS] = $body;
        }
        curl_setopt_array($ch, $opts);
        $response = curl_exec($ch);
        if ($response === false) {
            $err = curl_error($ch);
            curl_close($ch);
            return [502, json_encode(['error' => ['code' => 'UPSTREAM_ERROR', 'message' => $err !== '' ? $err : 'Upstream request failed']])];
        }
        $statusCode = (int) curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);
        return [$statusCode > 0 ? $statusCode : 502, (string) $response];
    }

    $httpOptions = [
        'method' => $method,
        'header' => implode("\r\n", $headers),
        'timeout' => $timeoutSeconds,
        'ignore_errors' => true,
    ];
    if ($body !== null) {
        $httpOptions['content'] = $body;
    }
    $context = stream_context_create([
        'http' => $httpOptions,
        'ssl' => [
            'verify_peer' => true,
            'verify_peer_name' => true,
        ],
    ]);

    $response = @file_get_contents($url, false, $context);
    $statusCode = 502;
    if (isset($http_response_header) && is_array($http_response_header) && isset($http_response_header[0])) {
        if (preg_match('/HTTP\/\S+\s+(\d{3})/', $http_response_header[0], $matches)) {
            $statusCode = (int) $matches[1];
        }
    }
    if ($response === false) {
        return [502, json_encode(['error' => ['code' => 'UPSTREAM_ERROR', 'message' => 'Upstream request failed']])];
    }
    return [$statusCode, (string) $response];
}

$method = strtoupper((string) ($_SERVER['REQUEST_METHOD'] ?? 'GET'));
$action = (string) ($_GET['action'] ?? '');

switch ($action) {
    case 'models':
        if ($method !== 'GET') {
            respondJsonError(405, 'METHOD_NOT_ALLOWED', 'GET method required');
        }
        [$status, $responseBody] = forwardRequest(
            'GET',
            'https://api.1min.ai/models?feature=UNIFY_CHAT_WITH_AI',
            ['Accept: application/json'],
            null,
            15
        );
        http_response_code($status);
        echo $responseBody;
        break;

    case 'validate':
        if ($method !== 'GET') {
            respondJsonError(405, 'METHOD_NOT_ALLOWED', 'GET method required');
        }
        $apiKey = extractApiKey();
        if ($apiKey === '') {
            http_response_code(401);
            echo json_encode(['errorCode' => 'MISSING_API_KEY', 'message' => 'API key is required']);
            exit;
        }
        [$status, $responseBody] = forwardRequest(
            'GET',
            'https://api.1min.ai/openai/v1/models',
            [
                'Accept: application/json',
                'Authorization: Bearer ' . $apiKey,
            ],
            null,
            15
        );
        http_response_code($status);
        echo $responseBody;
        break;

    case 'chat':
        if ($method !== 'POST') {
            respondJsonError(405, 'METHOD_NOT_ALLOWED', 'POST method required');
        }
        $apiKey = extractApiKey();
        if ($apiKey === '') {
            http_response_code(401);
            echo json_encode(['errorCode' => 'MISSING_API_KEY', 'message' => 'API key is required']);
            exit;
        }
        $rawBody = file_get_contents('php://input');
        if (!is_string($rawBody) || trim($rawBody) === '') {
            respondJsonError(400, 'INVALID_REQUEST', 'Request body is required');
        }
        json_decode($rawBody, true);
        if (json_last_error() !== JSON_ERROR_NONE) {
            respondJsonError(400, 'INVALID_JSON', 'Request body must be valid JSON');
        }
        [$status, $responseBody] = forwardRequest(
            'POST',
            'https://api.1min.ai/api/chat-with-ai',
            [
                'Content-Type: application/json',
                'Accept: application/json',
                'API-KEY: ' . $apiKey,
            ],
            $rawBody,
            90
        );
        http_response_code($status);
        echo $responseBody;
        break;

    default:
        respondJsonError(400, 'INVALID_ACTION', 'Unsupported action');
}
