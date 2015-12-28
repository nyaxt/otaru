'use strict';

// Scan Your HTML For Assets & Optimize Them
module.exports = function (gulp, plugins, config) { return function () {
  return gulp.src(['app/**/*.html', '.tmp/*.html', 'app/{elements,test}/**/*.html'])
    .pipe(plugins.useref({searchPath: ['.tmp', 'app', 'dist']}))
    // Concatenate And Minify JavaScript
    .pipe(plugins.if('*.js', plugins.uglify({preserveComments: 'some'})))
    // Concatenate And Minify Styles
    // In case you are still using useref build blocks
    .pipe(plugins.if('*.css', plugins.cssmin()))
    // Add shim-shadowdom to link with main.css
    .pipe(plugins.if('*.html', plugins.replace(
      'main.css">', 'main.css" shim-shadowdom>')))
    // Minify Any HTML
    .pipe(plugins.if('*.html', plugins.htmlmin({
      removeEmptyAttributes: false,
      collapseWhiteSpace: true,
      conservativeCollapse: true,
      removeAttributeQuotes: false,
      removeRedundantAttributes: false,
      customAttrAssign: [/\?=/, /\$=/],
    })))
    // Output Files
    .pipe(gulp.dest('dist'))
    .pipe(plugins.size({title: 'html'}));
};};
