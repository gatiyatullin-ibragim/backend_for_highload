package com.example.practice1;

import org.springframework.web.bind.annotation.RestController;
import java.util.map;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;



@RestController
@RequestMapping("/api")
public class TestController {

    @GetMapping("/work")
    public Map<String, Object> doWork(@RequestParam(defaultValue = "0") int delayMs) throws InterruptedException{
        if (delayMs > 0){
            Thread.sleep(delayMs);
        }
        return Map.of(
            "status", "ok",
            "thread", Thread.currentThread().getName()
        );
    }
    
}
