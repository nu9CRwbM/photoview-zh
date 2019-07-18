import React, { Component } from 'react'
import { createGlobalStyle } from 'styled-components'

const GlobalStyle = createGlobalStyle`
  #root, body {
    height: 100%;
    margin: 0;
  }
`

import Routes from './Routes'

class App extends Component {
  render() {
    return (
      <>
        <GlobalStyle />
        <Routes />
      </>
    )
  }
}

export default App