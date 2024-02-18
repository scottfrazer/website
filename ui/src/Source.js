import SyntaxHighlighter from 'react-syntax-highlighter';
import { solarizedDark } from 'react-syntax-highlighter/dist/esm/styles/hljs';

const Source = () => {
  const codeString = `package main

import "fmt"

func main() {
  fmt.Println("Coming soon...")
}
    `;
  return (
    <div className="code-block">
      <SyntaxHighlighter language="go" style={solarizedDark}>
        {codeString}
      </SyntaxHighlighter>
    </div>
  );
};

export default Source