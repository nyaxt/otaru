'use strict';

module.exports = function (gulp, plugins, config) { return function () {
  return gulp.src([
      'app/scripts/**/*.js',
      'app/elements/**/*.js',
      'app/elements/**/*.html'
    ])
    .pipe(plugins.sourcemaps.init())
    .pipe(plugins.babel({ nonStandard: false }))
    .pipe(plugins.sourcemaps.write('dist/scripts'))
    // Output Files
    .pipe(gulp.dest('dist'));
};};
