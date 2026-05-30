import os
import re

directory = 'backend'

# regex to find: func (m *MockMarketService) GetDividends(ctx context.Context, ticker string) ([]market.DividendEvent, error) {
# and replace with: func (m *MockMarketService) GetDividends(ctx context.Context, ticker string, assetType string) ([]market.DividendEvent, error) {
# Note: some are []DividendEvent and some are []market.DividendEvent depending on the package.
# Some mocks are MockQuoteProvider, MockMarketProvider, etc.

pattern = re.compile(r'func \((.*?)\) GetDividends\(ctx context\.Context, (\w+) string\) \(\[\](.*?)DividendEvent, error\) \{')

for root, dirs, files in os.walk(directory):
    for file in files:
        if file.endswith('_test.go'):
            filepath = os.path.join(root, file)
            with open(filepath, 'r') as f:
                content = f.read()
            
            new_content = pattern.sub(r'func (\1) GetDividends(ctx context.Context, \2 string, assetType string) ([]\3DividendEvent, error) {', content)
            
            if new_content != content:
                with open(filepath, 'w') as f:
                    f.write(new_content)
                print(f"Fixed {filepath}")
