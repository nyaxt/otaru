'use strict';

module.exports = function (gulp, plugins, config) { return function () {
  return require('merge-stream')(
    gulp.src('app/scripts/**/*.js')
      .pipe(plugins.babel())
      .pipe(gulp.dest('dist/scripts'))
  ).pipe(plugins.size({title: 'babel'}));
};};
