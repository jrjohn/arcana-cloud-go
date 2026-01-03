#!/bin/bash
# Comprehensive Test Report Generator for Arcana Cloud Go
# Generates HTML coverage reports and test summaries

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Directories
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCS_DIR="$PROJECT_ROOT/docs"
COVERAGE_DIR="$DOCS_DIR/coverage"
REPORTS_DIR="$DOCS_DIR/test-reports"
TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")

# Create directories
mkdir -p "$COVERAGE_DIR"
mkdir -p "$REPORTS_DIR"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   Arcana Cloud Go Test Report Generator${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Function to run tests with coverage
run_tests() {
    echo -e "${YELLOW}Running tests with coverage...${NC}"

    # Run tests and generate coverage profile
    go test -v -race -coverprofile="$COVERAGE_DIR/coverage.out" -covermode=atomic ./... 2>&1 | tee "$REPORTS_DIR/test-output.txt"

    # Check if tests passed
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        echo -e "${GREEN}All tests passed!${NC}"
        return 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        return 1
    fi
}

# Function to generate HTML coverage report
generate_html_coverage() {
    echo -e "${YELLOW}Generating HTML coverage report...${NC}"

    go tool cover -html="$COVERAGE_DIR/coverage.out" -o "$COVERAGE_DIR/coverage.html"

    # Generate per-package coverage
    go tool cover -func="$COVERAGE_DIR/coverage.out" > "$COVERAGE_DIR/coverage-func.txt"

    echo -e "${GREEN}HTML coverage report generated: $COVERAGE_DIR/coverage.html${NC}"
}

# Function to calculate coverage percentage
calculate_coverage() {
    local total_coverage
    total_coverage=$(go tool cover -func="$COVERAGE_DIR/coverage.out" | grep "total:" | awk '{print $3}')
    echo "$total_coverage"
}

# Function to generate summary report
generate_summary_report() {
    echo -e "${YELLOW}Generating summary report...${NC}"

    local coverage_pct
    coverage_pct=$(calculate_coverage)

    local test_count
    test_count=$(grep -c "=== RUN" "$REPORTS_DIR/test-output.txt" 2>/dev/null || echo "0")

    local pass_count
    pass_count=$(grep -c "--- PASS:" "$REPORTS_DIR/test-output.txt" 2>/dev/null || echo "0")

    local fail_count
    fail_count=$(grep -c "--- FAIL:" "$REPORTS_DIR/test-output.txt" 2>/dev/null || echo "0")

    local skip_count
    skip_count=$(grep -c "--- SKIP:" "$REPORTS_DIR/test-output.txt" 2>/dev/null || echo "0")

    # Generate HTML summary
    cat > "$REPORTS_DIR/summary.html" << EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Arcana Cloud Go - Test Report</title>
    <style>
        :root {
            --primary: #6366f1;
            --success: #22c55e;
            --warning: #eab308;
            --danger: #ef4444;
            --bg: #0f172a;
            --card-bg: #1e293b;
            --text: #f8fafc;
            --text-muted: #94a3b8;
        }
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Segoe UI', system-ui, sans-serif;
            background: var(--bg);
            color: var(--text);
            min-height: 100vh;
            padding: 2rem;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        header {
            text-align: center;
            margin-bottom: 3rem;
        }
        h1 {
            font-size: 2.5rem;
            background: linear-gradient(135deg, var(--primary), #8b5cf6);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 0.5rem;
        }
        .timestamp {
            color: var(--text-muted);
            font-size: 0.875rem;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1.5rem;
            margin-bottom: 3rem;
        }
        .stat-card {
            background: var(--card-bg);
            border-radius: 1rem;
            padding: 1.5rem;
            text-align: center;
            border: 1px solid rgba(255,255,255,0.1);
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .stat-card:hover {
            transform: translateY(-4px);
            box-shadow: 0 20px 40px rgba(0,0,0,0.3);
        }
        .stat-value {
            font-size: 3rem;
            font-weight: 700;
            margin-bottom: 0.5rem;
        }
        .stat-label {
            color: var(--text-muted);
            text-transform: uppercase;
            font-size: 0.75rem;
            letter-spacing: 0.1em;
        }
        .coverage-value { color: var(--success); }
        .pass-value { color: var(--success); }
        .fail-value { color: var(--danger); }
        .skip-value { color: var(--warning); }
        .coverage-bar-container {
            background: var(--card-bg);
            border-radius: 1rem;
            padding: 2rem;
            margin-bottom: 2rem;
            border: 1px solid rgba(255,255,255,0.1);
        }
        .coverage-bar {
            background: rgba(255,255,255,0.1);
            border-radius: 9999px;
            height: 2rem;
            overflow: hidden;
            position: relative;
        }
        .coverage-fill {
            background: linear-gradient(90deg, var(--primary), var(--success));
            height: 100%;
            border-radius: 9999px;
            transition: width 1s ease-out;
            display: flex;
            align-items: center;
            justify-content: flex-end;
            padding-right: 1rem;
            font-weight: 600;
        }
        .section {
            background: var(--card-bg);
            border-radius: 1rem;
            padding: 2rem;
            margin-bottom: 2rem;
            border: 1px solid rgba(255,255,255,0.1);
        }
        .section h2 {
            font-size: 1.25rem;
            margin-bottom: 1rem;
            color: var(--primary);
        }
        .package-list {
            list-style: none;
        }
        .package-item {
            display: flex;
            justify-content: space-between;
            padding: 0.75rem 0;
            border-bottom: 1px solid rgba(255,255,255,0.05);
        }
        .package-item:last-child {
            border-bottom: none;
        }
        .package-name {
            font-family: 'Consolas', monospace;
            font-size: 0.875rem;
        }
        .package-coverage {
            font-weight: 600;
        }
        .links {
            display: flex;
            gap: 1rem;
            justify-content: center;
            margin-top: 2rem;
        }
        .link-btn {
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
            background: var(--primary);
            color: white;
            padding: 0.75rem 1.5rem;
            border-radius: 0.5rem;
            text-decoration: none;
            font-weight: 500;
            transition: opacity 0.2s;
        }
        .link-btn:hover {
            opacity: 0.9;
        }
        footer {
            text-align: center;
            margin-top: 3rem;
            color: var(--text-muted);
            font-size: 0.875rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>Arcana Cloud Go</h1>
            <p class="timestamp">Test Report - $TIMESTAMP</p>
        </header>

        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-value coverage-value">$coverage_pct</div>
                <div class="stat-label">Code Coverage</div>
            </div>
            <div class="stat-card">
                <div class="stat-value pass-value">$pass_count</div>
                <div class="stat-label">Tests Passed</div>
            </div>
            <div class="stat-card">
                <div class="stat-value fail-value">$fail_count</div>
                <div class="stat-label">Tests Failed</div>
            </div>
            <div class="stat-card">
                <div class="stat-value skip-value">$skip_count</div>
                <div class="stat-label">Tests Skipped</div>
            </div>
        </div>

        <div class="coverage-bar-container">
            <h3 style="margin-bottom: 1rem;">Overall Coverage</h3>
            <div class="coverage-bar">
                <div class="coverage-fill" style="width: $coverage_pct;">$coverage_pct</div>
            </div>
        </div>

        <div class="section">
            <h2>Package Coverage Details</h2>
            <ul class="package-list">
$(grep -v "total:" "$COVERAGE_DIR/coverage-func.txt" 2>/dev/null | grep -v "^$" | head -30 | while read line; do
    pkg=$(echo "$line" | awk '{print $1}')
    cov=$(echo "$line" | awk '{print $NF}')
    echo "                <li class=\"package-item\"><span class=\"package-name\">$pkg</span><span class=\"package-coverage\">$cov</span></li>"
done)
            </ul>
        </div>

        <div class="links">
            <a href="../coverage/coverage.html" class="link-btn">
                <span>View Full Coverage Report</span>
            </a>
        </div>

        <footer>
            <p>Generated with Arcana Cloud Go Test Suite</p>
        </footer>
    </div>
</body>
</html>
EOF

    echo -e "${GREEN}Summary report generated: $REPORTS_DIR/summary.html${NC}"
}

# Function to generate badge
generate_badges() {
    echo -e "${YELLOW}Generating coverage badges...${NC}"

    local coverage_pct
    coverage_pct=$(calculate_coverage | tr -d '%')

    local color="red"
    if (( $(echo "$coverage_pct >= 80" | bc -l) )); then
        color="brightgreen"
    elif (( $(echo "$coverage_pct >= 60" | bc -l) )); then
        color="green"
    elif (( $(echo "$coverage_pct >= 40" | bc -l) )); then
        color="yellow"
    elif (( $(echo "$coverage_pct >= 20" | bc -l) )); then
        color="orange"
    fi

    echo "Coverage: $coverage_pct% ($color)"
}

# Function to print final summary
print_summary() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}           Report Generation Complete   ${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo -e "${GREEN}Generated Reports:${NC}"
    echo "  - HTML Coverage: $COVERAGE_DIR/coverage.html"
    echo "  - Coverage Functions: $COVERAGE_DIR/coverage-func.txt"
    echo "  - Coverage Profile: $COVERAGE_DIR/coverage.out"
    echo "  - Test Summary: $REPORTS_DIR/summary.html"
    echo "  - Test Output: $REPORTS_DIR/test-output.txt"
    echo ""

    local coverage_pct
    coverage_pct=$(calculate_coverage)
    echo -e "${GREEN}Total Coverage: ${coverage_pct}${NC}"
    echo ""
    echo -e "Open ${BLUE}$REPORTS_DIR/summary.html${NC} in your browser to view the full report."
}

# Main execution
main() {
    cd "$PROJECT_ROOT"

    # Run tests (continue even if some fail)
    run_tests || true

    # Generate reports if coverage file exists
    if [ -f "$COVERAGE_DIR/coverage.out" ]; then
        generate_html_coverage
        generate_summary_report
        generate_badges
    else
        echo -e "${RED}No coverage data generated. Check test output.${NC}"
        exit 1
    fi

    print_summary
}

main "$@"
