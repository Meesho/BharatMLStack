package com.bharatmlstack.service;

import com.bharatmlstack.BharatMLClient;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class ClientConfig {
    @Bean
    public BharatMLClient bharatMLClient() {
        return new BharatMLClient("online-feature-store-api.int.meesho.int", 80);
    }
}
