import SyntaxHighlighter from 'react-syntax-highlighter';
import { solarizedDark } from 'react-syntax-highlighter/dist/esm/styles/hljs';

const Source = (props) => {
  return (
    <div className="code-block">
      <SyntaxHighlighter language="go" style={solarizedDark}>
        {props.code}
      </SyntaxHighlighter>
    </div>
  );
};

export default Source