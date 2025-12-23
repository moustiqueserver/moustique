package main

import (
	"fmt"
	"os"
	"strings"
	"slices"
)

var topicExplosionCache = make(map[string][]string)

func explodeTopic(topic string) []string {
	if cached, exists := topicExplosionCache[topic]; exists {
		return cached
	}

	var patterns []string
	sections := strings.Split(topic, "/")

	// Loopa från sista elementet ner till index 1
	for i := len(sections) - 1; i >= 1; i-- {
		// Huvudmönster: map { $_ eq $sections[$i] ? $_ : '+' } @sections[$i..$#sections]
		// Jämför varje element från i till slutet med sections[i] själv
		beforeI := sections[:i]
		fromI := sections[i:]
		targetValue := sections[i] // Detta är nyckeln!
		
		// Ersätt alla element i fromI som INTE är lika med targetValue till +
		mappedFromI := make([]string, len(fromI))
		for j, sec := range fromI {
			if sec == targetValue {
				mappedFromI[j] = sec
			} else {
				mappedFromI[j] = "+"
			}
		}
		
		patternParts := make([]string, 0, len(sections))
		patternParts = append(patternParts, beforeI...)
		patternParts = append(patternParts, mappedFromI...)
		pattern := strings.Join(patternParts, "/")
		patterns = append(patterns, pattern)

		// "Insprängt wildcard": sätt in + FÖRE position i
		// if($i>2 && $i <= $#sections)
		if i > 2 && i <= len(sections)-1 {
			beforeIMinus1 := sections[:i-1]
			fromI := sections[i:]
			
			insprangtParts := make([]string, 0, len(sections)+1)
			insprangtParts = append(insprangtParts, beforeIMinus1...)
			insprangtParts = append(insprangtParts, "+")
			insprangtParts = append(insprangtParts, fromI...)
			insprangt := strings.Join(insprangtParts, "/")
			if !slices.Contains(patterns, insprangt) {
			  patterns = append(patterns, insprangt)
		        } else {
			  fmt.Printf("%s fanns redan, lagger inte till",insprangt)
		        }
		}
	}

	// Lägg INTE till original topic här - den ska vara först!
	// Perl pushar in i början, så vi måste prependa
	result := []string{} //{topic}
	result = append(result, patterns...)

	topicExplosionCache[topic] = result
	return result
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: test_explode <topic> [subscription]")
		fmt.Println("Example: test_explode /mushroom/logs/moustique_lib/INFO '/mushroom/logs/+/+'")
		os.Exit(1)
	}

	topic := os.Args[1]
	patterns := explodeTopic(topic)

	fmt.Printf("Topic: %s\n", topic)
	fmt.Printf("Generated %d patterns:\n", len(patterns))
	for i, pattern := range patterns {
		fmt.Printf("  %d: %s\n", i+1, pattern)
	}

	// Om användaren ger ett andra argument, matcha mot det
	if len(os.Args) >= 3 {
		subscription := os.Args[2]
		fmt.Printf("\nChecking if subscription '%s' would match:\n", subscription)
		matched := false
		for _, pattern := range patterns {
			if pattern == subscription {
				fmt.Printf("  ✓ YES - pattern '%s' matches subscription\n", pattern)
				matched = true
			}
		}
		if !matched {
			fmt.Printf("  ✗ NO - subscription not in generated patterns\n")
			
			// Visa vilka patterns som genererades
			fmt.Println("\nGenerated patterns were:")
			for _, p := range patterns {
				fmt.Printf("  - %s\n", p)
			}
		}
	}
}
