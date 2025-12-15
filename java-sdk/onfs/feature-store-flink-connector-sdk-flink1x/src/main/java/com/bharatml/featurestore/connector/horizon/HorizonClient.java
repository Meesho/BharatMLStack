package com.bharatml.featurestore.connector.horizon;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;

public final class HorizonClient {

    private static final Logger logger = LoggerFactory.getLogger(HorizonClient.class);

    private final String horizonBaseUrl;
    private final HttpClient httpClient;
    private final ObjectMapper objectMapper;

    public HorizonClient(String horizonBaseUrl) {
        this.horizonBaseUrl = horizonBaseUrl;

        this.httpClient = HttpClient.newBuilder()
                .connectTimeout(Duration.ofSeconds(30))
                .build();

        this.objectMapper = new ObjectMapper();
    }

    public SourceMappingResponse getHorizonResponse(String jobId, String jobToken)
            throws IOException, InterruptedException {

        String url = String.format(
                "%s/api/v1/orion/get-source-mapping?jobId=%s&jobToken=%s",
                horizonBaseUrl, jobId, jobToken
        );

        logger.info("Sending Horizon request to {}", url);

        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(url))
                .POST(HttpRequest.BodyPublishers.noBody())
                .timeout(Duration.ofSeconds(15))
                // DO NOT set Accept-Encoding → avoids gzip issues
                .header("Accept", "application/json")
                .build();

        HttpResponse<String> response =
                httpClient.send(request, HttpResponse.BodyHandlers.ofString());

        int status = response.statusCode();
        String body = response.body();

        logger.info("Horizon response status = {}", status);

        if (status != 200) {
            logger.warn("Horizon returned non-200 ({}) body: {}", status, body);
            throw new IOException("Horizon API error " + status + ": " + body);
        }

        try {
            return objectMapper.readValue(body, SourceMappingResponse.class);
        } catch (Exception e) {
            logger.error("Failed to parse Horizon JSON → {}", e.getMessage());
            logger.debug("Raw body returned: {}", body);
            throw new IOException("Invalid JSON returned by Horizon", e);
        }
    }
}
