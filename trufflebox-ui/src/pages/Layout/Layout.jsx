import React from 'react';
import Header from '../Header/index';
import PropTypes from 'prop-types';

const Layout = ({ children }) => {
  return (
    <div>
      <Header /> {/* Persistent Header */}
      <main>{children}</main> {/* Render the page content */}
    </div>
  );
};

Layout.propTypes = {
  children: PropTypes.node.isRequired,
};

export default Layout;
