package com.example.practice2;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.stream.LongStream;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;


record Quote(long id, String pair, double bid){}

@RestController 
@RequestMapping("/api")
public class QuoteController {
    
    @GetMapping("/quote/{id}")
    public Quote one(@PathVariable long id) {
        return new Quote(id, "USDTKZ", 533.14);
    }

    @GetMapping("/qoutes")
    public List<Quote> many(@RequestParam(defaultValue = "1") int n){
        return LongStream.rangeClosed(1, n)
                .mapToObj(i -> new Quote(i, "USDTKZ", 533.14))
                .toList();
    }
    
}
